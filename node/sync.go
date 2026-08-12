package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/ledger"
)

var syncHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

type SyncResult struct {
	Adopted      bool
	LocalHeight  int
	RemoteHeight int
	Reason       string
}

func FetchPeerChain(peer string) ([]block.Block, error) {
	url := fmt.Sprintf("http://%s/chain", peer)

	resp, err := syncHTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to fetch chain from peer %s: %w",
			peer,
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"peer %s returned HTTP status %d",
			peer,
			resp.StatusCode,
		)
	}

	var blocks []block.Block

	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return nil, fmt.Errorf(
			"failed to decode chain from peer %s: %w",
			peer,
			err,
		)
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf(
			"peer %s returned an empty chain",
			peer,
		)
	}

	return blocks, nil
}

func ValidateCandidateChain(blocks []block.Block) error {
	if len(blocks) == 0 {
		return fmt.Errorf("candidate chain is empty")
	}

	candidate := &chain.Blockchain{
		Blocks: blocks,
	}

	valid, reason := candidate.ValidateChain()

	if !valid {
		return fmt.Errorf(
			"candidate chain is invalid: %s",
			reason,
		)
	}

	return nil
}

func (n *Node) SyncFromPeer(peer string) (SyncResult, error) {
	remoteBlocks, err := FetchPeerChain(peer)
	if err != nil {
		return SyncResult{}, err
	}

	if err := ValidateCandidateChain(remoteBlocks); err != nil {
		return SyncResult{
			Adopted: false,
			Reason:  "invalid peer chain",
		}, err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	localHeight := n.Blockchain.GetLatestBlock().Index
	remoteHeight := remoteBlocks[len(remoteBlocks)-1].Index

	if chainsEqual(n.Blockchain.Blocks, remoteBlocks) {
		return SyncResult{
			Adopted:      false,
			LocalHeight:  localHeight,
			RemoteHeight: remoteHeight,
			Reason:       "chains are identical",
		}, nil
	}

	// Snapshot the current chain before ResolveFork potentially
	// overwrites n.Blockchain.Blocks in place, so we can recover
	// transactions from any blocks that end up orphaned by the swap.
	oldBlocks := make([]block.Block, len(n.Blockchain.Blocks))
	copy(oldBlocks, n.Blockchain.Blocks)

	candidate := &chain.Blockchain{
		Blocks: remoteBlocks,
	}

	adopted, reason := n.Blockchain.ResolveFork(candidate)

	if !adopted {
		return SyncResult{
			Adopted:      false,
			LocalHeight:  localHeight,
			RemoteHeight: remoteHeight,
			Reason:       reason,
		}, nil
	}

	// Recover transactions from orphaned blocks. reconcilePendingLocked
	// only re-validates what's already in n.Pending — but a mined
	// block's transactions were already stripped out of Pending when
	// that block was accepted (see removeMinedTransactionsLocked). If
	// that block is now orphaned by the chain swap above, its
	// transactions must be re-queued here before reconciliation runs,
	// or they're silently lost rather than returned to the pool per
	// FR-6.
	newHashes := make(map[string]bool, len(n.Blockchain.Blocks))

	for _, b := range n.Blockchain.Blocks {
		newHashes[b.Hash] = true
	}

	for _, b := range oldBlocks {

		if newHashes[b.Hash] {
			continue
		}

		n.Pending = append(n.Pending, b.Transactions...)
	}

	n.reconcilePendingLocked()

	for _, b := range n.Blockchain.Blocks {
		n.seenBlocks[b.Hash] = true
	}

	return SyncResult{
		Adopted:      true,
		LocalHeight:  localHeight,
		RemoteHeight: remoteHeight,
		Reason:       reason,
	}, nil
}

func (n *Node) SyncFromPeers() []SyncResult {
	peers := n.PeerList()

	results := make([]SyncResult, 0, len(peers))

	for _, peer := range peers {
		if peer == n.Address {
			continue
		}

		result, err := n.SyncFromPeer(peer)

		if err != nil {
			result.Reason = err.Error()
		}

		results = append(results, result)
	}

	return results
}

func (n *Node) reconcilePendingLocked() {
	confirmed := make(map[string]bool)

	for _, b := range n.Blockchain.Blocks {
		for _, tx := range b.Transactions {
			if tx.Signature != "" {
				confirmed[tx.Signature] = true
			}
		}
	}

	ld := n.Blockchain.BuildLedger()

	remaining := make(
		[]ledger.Transaction,
		0,
		len(n.Pending),
	)

	for _, tx := range n.Pending {

		if tx.Signature != "" && confirmed[tx.Signature] {
			continue
		}

		if err := ld.ApplyTransaction(tx); err != nil {

			continue
		}

		remaining = append(remaining, tx)
	}

	n.Pending = remaining

	n.seenTx = make(map[string]bool)

	for _, b := range n.Blockchain.Blocks {
		for _, tx := range b.Transactions {
			if tx.Signature != "" {
				n.seenTx[tx.Signature] = true
			}
		}
	}

	for _, tx := range n.Pending {
		if tx.Signature != "" {
			n.seenTx[tx.Signature] = true
		}
	}
}

func chainsEqual(a, b []block.Block) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Hash != b[i].Hash {
			return false
		}
	}

	return true
}
