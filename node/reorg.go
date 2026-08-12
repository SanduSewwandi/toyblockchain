package node

import (
	"log"

	"toyblockchain/block"
)

// HandleUnexpectedBlock is called when AddBlock (block.go) reports that
// a received block does not extend the current chain tip — either this
// node has fallen behind, or the sender has mined a competing block at
// the same height, causing a fork.
//
// It does not attempt to resolve the fork from the single block alone;
// a single block can't represent a full competing chain. Instead it
// asks the source peer for its complete chain and lets SyncFromPeer
// (sync.go) validate it and apply chain.ResolveFork's rule.
func (n *Node) HandleUnexpectedBlock(b block.Block, fromPeer string) {

	if fromPeer == "" {
		// No peer to sync from (e.g. block arrived with no origin
		// header) — nothing actionable to do here.
		return
	}

	result, err := n.SyncFromPeer(fromPeer)

	if err != nil {
		log.Printf(
			"reorg check with %s failed: %v",
			fromPeer,
			err,
		)
		return
	}

	if result.Adopted {
		log.Printf(
			"chain reorganized: adopted %d-block chain from %s (was %d blocks): %s",
			result.RemoteHeight+1,
			fromPeer,
			result.LocalHeight+1,
			result.Reason,
		)
		return
	}

	log.Printf(
		"received block did not extend our chain and %s's chain was not adopted: %s",
		fromPeer,
		result.Reason,
	)
}
