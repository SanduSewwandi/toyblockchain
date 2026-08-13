package node

import (
	"fmt"

	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/ledger"
)

// AddBlockResult describes what happened when a node received a block.
type AddBlockResult struct {
	Accepted bool
	Reason   string
}

func (n *Node) AddBlock(b block.Block) (AddBlockResult, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Ignore blocks that this node has already accepted.
	if n.seenBlocks[b.Hash] {
		return AddBlockResult{
			Accepted: false,
			Reason:   "block already seen",
		}, nil
	}

	if err := validateReceivedBlock(b); err != nil {
		return AddBlockResult{
			Accepted: false,
			Reason:   "invalid block",
		}, fmt.Errorf("invalid block: %w", err)
	}

	latest := n.Blockchain.GetLatestBlock()

	if b.Index == latest.Index+1 &&
		b.PreviousHash == latest.Hash {

		
		expectedDifficulty := chain.NextDifficultyFor(
			n.Blockchain,
			chain.DefaultDifficulty,
		)

		if b.Difficulty != expectedDifficulty {
			return AddBlockResult{
					Accepted: false,
					Reason:   "invalid difficulty",
				}, fmt.Errorf(
					"block difficulty %d does not match expected difficulty %d",
					b.Difficulty,
					expectedDifficulty,
				)
		}

		if b.Timestamp < latest.Timestamp {
			return AddBlockResult{
					Accepted: false,
					Reason:   "invalid timestamp",
				}, fmt.Errorf(
					"block timestamp %d precedes previous block timestamp %d",
					b.Timestamp,
					latest.Timestamp,
				)
		}

		ld := n.Blockchain.BuildLedger()

		if err := ld.ApplyBlockTransactions(b.Transactions); err != nil {
			return AddBlockResult{
					Accepted: false,
					Reason:   "invalid block transactions",
				}, fmt.Errorf(
					"block transactions rejected: %w",
					err,
				)
		}

		// Append the already-mined block.
		n.Blockchain.Blocks = append(
			n.Blockchain.Blocks,
			b,
		)

		// Only mark as seen once it's actually accepted onto the chain,
		// so a block that fails to extend the tip remains re-processable
		// for a later sync/reorg.
		n.seenBlocks[b.Hash] = true

		// Remove transactions that are now confirmed.
		n.removeMinedTransactionsLocked(b.Transactions)

		return AddBlockResult{
			Accepted: true,
			Reason:   "block appended to current chain",
		}, nil
	}

	return AddBlockResult{
		Accepted: false,
		Reason:   "block does not extend current chain",
	}, nil
}

func validateReceivedBlock(b block.Block) error {
	// Recalculate the block hash and compare it with the stored hash.
	expectedHash := b.CalculateHash()

	if b.Hash != expectedHash {
		return fmt.Errorf(
			"hash mismatch: expected %s, got %s",
			expectedHash,
			b.Hash,
		)
	}

	if b.Difficulty < chain.MinDifficulty {
		return fmt.Errorf(
			"difficulty %d below minimum %d",
			b.Difficulty,
			chain.MinDifficulty,
		)
	}

	// Verify proof-of-work.
	if len(b.Hash) < b.Difficulty {
		return fmt.Errorf("hash is shorter than difficulty")
	}

	for i := 0; i < b.Difficulty; i++ {
		if b.Hash[i] != '0' {
			return fmt.Errorf(
				"proof-of-work failed: hash %s does not have %d leading zeroes",
				b.Hash,
				b.Difficulty,
			)
		}
	}

	// Verify the Merkle root against the transactions contained in
	// the received block.
	expectedMerkleRoot := block.MerkleRoot(b.Transactions)

	if b.MerkleRoot != expectedMerkleRoot {
		return fmt.Errorf(
			"merkle root mismatch: expected %s, got %s",
			expectedMerkleRoot,
			b.MerkleRoot,
		)
	}

	// Enforce the configured maximum block size.
	if len(b.Transactions) > chain.DefaultBlockSize {
		return fmt.Errorf(
			"block contains %d transactions, maximum allowed is %d",
			len(b.Transactions),
			chain.DefaultBlockSize,
		)
	}

	return nil
}

func (n *Node) RemoveMinedTransactions(
	mined []ledger.Transaction,
) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.removeMinedTransactionsLocked(mined)
}
