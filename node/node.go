package node

import (
	"sync"

	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/ledger"
)

type Node struct {
	mu sync.RWMutex

	Blockchain *chain.Blockchain
	Pending    []ledger.Transaction
	Peers      map[string]bool // peer address -> known

	seenTx     map[string]bool
	seenBlocks map[string]bool

	Address string
}

// NewNode creates a node with a fresh genesis blockchain and the given
// listen address and initial peer list.
func NewNode(address string, initialPeers []string) *Node {

	peers := make(map[string]bool)

	for _, p := range initialPeers {
		peers[p] = true
	}

	return &Node{
		Blockchain: chain.NewBlockchain(),
		Pending:    []ledger.Transaction{},
		Peers:      peers,
		Address:    address,
		seenTx:     make(map[string]bool),
		seenBlocks: make(map[string]bool),
	}
}

// Height returns the current chain height (highest block index).
func (n *Node) Height() int {

	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.Blockchain.GetLatestBlock().Index
}

// HeadHash returns the hash of the current chain's latest block.
func (n *Node) HeadHash() string {

	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.Blockchain.GetLatestBlock().Hash
}

// PeerList returns a snapshot slice of known peer addresses.
func (n *Node) PeerList() []string {

	n.mu.RLock()
	defer n.mu.RUnlock()

	list := make([]string, 0, len(n.Peers))

	for p := range n.Peers {
		list = append(list, p)
	}

	return list
}

// PendingCount returns the number of transactions waiting to be mined.
func (n *Node) PendingCount() int {

	n.mu.RLock()
	defer n.mu.RUnlock()

	return len(n.Pending)
}

// Balance returns the ledger balance for a given address, rebuilt from
// the current chain.
func (n *Node) Balance(address string) int64 {

	n.mu.RLock()
	defer n.mu.RUnlock()

	ld := n.Blockchain.BuildLedger()
	return ld.GetBalance(address)
}

// ChainSnapshot returns a copy of the current blockchain's blocks, safe
// to serialize for a /chain endpoint without holding the lock during I/O.
func (n *Node) ChainSnapshot() []block.Block {

	n.mu.RLock()
	defer n.mu.RUnlock()

	snapshot := make([]block.Block, len(n.Blockchain.Blocks))
	copy(snapshot, n.Blockchain.Blocks)

	return snapshot
}

func (n *Node) BlockAt(index int) (block.Block, bool) {

	n.mu.RLock()
	defer n.mu.RUnlock()

	if index < 0 || index >= len(n.Blockchain.Blocks) {
		return block.Block{}, false
	}

	return n.Blockchain.Blocks[index], true
}
