package node

import (
	"sync"

	"toyblockchain/chain"
	"toyblockchain/ledger"
)


type Node struct {
	mu sync.RWMutex

	Blockchain *chain.Blockchain
	Pending    []ledger.Transaction
	Peers      map[string]bool // peer address -> known


	Address string
}


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
	}
}

// Height returns the current chain height (highest block index).
func (n *Node) Height() int {

	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.Blockchain.GetLatestBlock().Index
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