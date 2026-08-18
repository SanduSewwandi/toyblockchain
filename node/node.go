package node

import (
	"log"
	"sync"

	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/ledger"
)

type Node struct {
	// FR-7: mu guards Blockchain, Pending, Peers, seenTx, and
	// seenBlocks below. Every read or write to these fields — from
	// HTTP handlers, the mining loop, and gossip — must hold this
	// lock, so the suite passes go test -race ./...

	mu sync.RWMutex

	Blockchain *chain.Blockchain
	Pending    []ledger.Transaction
	Peers      map[string]bool // peer address -> known

	seenTx     map[string]bool
	seenBlocks map[string]bool

	Address string
}

// FR-1: Node is the networked service each process runs — one node,
// one address, its own peer list, so several instances can run side
// by side on different ports.
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

func (n *Node) AddPeer(addr string) {

	if addr == "" || addr == n.Address {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.Peers[addr] {
		n.Peers[addr] = true
		log.Printf("registered new peer %s (self-announced)", addr)
	}
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
