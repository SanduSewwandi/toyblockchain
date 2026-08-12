package node

import (
	"fmt"
	"log"
	"time"

	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/ledger"
)

func (n *Node) MineOnce() (minedBlock block.Block, ok bool, err error) {

	txs, prevHash, nextIndex, difficulty, hasWork := n.snapshotForMining()

	if !hasWork {
		return block.Block{}, false, nil
	}

	newBlock := block.NewBlock(nextIndex, txs, prevHash, difficulty)

	chain.MineBlockConcurrent(&newBlock, difficulty, chain.DefaultMiningWorkers)

	result, err := n.AddBlock(newBlock)

	if err != nil {
		return block.Block{}, false, fmt.Errorf("mined block rejected: %w", err)
	}

	if !result.Accepted {

		return block.Block{}, false, nil
	}

	return newBlock, true, nil
}

// snapshotForMining takes a read lock just long enough to copy what
// mining needs, so the actual proof-of-work loop runs lock-free.
func (n *Node) snapshotForMining() (txs []ledger.Transaction, prevHash string, nextIndex int, difficulty int, hasWork bool) {

	n.mu.RLock()
	defer n.mu.RUnlock()

	if len(n.Pending) == 0 {
		return nil, "", 0, 0, false
	}

	toMine := n.Pending

	if len(toMine) > chain.DefaultBlockSize {
		toMine = toMine[:chain.DefaultBlockSize]
	}

	txs = make([]ledger.Transaction, len(toMine))
	copy(txs, toMine)

	latest := n.Blockchain.GetLatestBlock()

	return txs,
		latest.Hash,
		latest.Index + 1,
		chain.NextDifficultyFor(n.Blockchain, chain.DefaultDifficulty),
		true
}

func (n *Node) StartMiningLoop(interval time.Duration, broadcast func(block.Block), stop <-chan struct{}) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {

		case <-stop:
			return

		case <-ticker.C:

			minedBlock, ok, err := n.MineOnce()

			if err != nil {
				log.Printf("mining error: %v", err)
				continue
			}

			if !ok {
				continue // nothing to mine, or the block went stale
			}

			hashPreview := minedBlock.Hash

			if len(hashPreview) > 8 {
				hashPreview = hashPreview[:8]
			}

			log.Printf(
				"mined block %d (hash %s...)",
				minedBlock.Index,
				hashPreview,
			)

			if broadcast != nil {
				broadcast(minedBlock)
			}
		}
	}
}
