package node

import (
	"toyblockchain/chain"
)

func (n *Node) SaveChainTo(path string) error {

	blocks := n.ChainSnapshot()

	snapshot := &chain.Blockchain{
		Blocks: blocks,
	}

	return snapshot.SaveToFile(path)
}
