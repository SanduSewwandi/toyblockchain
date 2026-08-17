package chain

import (
	"fmt"
	"math/big"

	"toyblockchain/block"
)


func blockWork(difficulty int) *big.Int {

	if difficulty < 0 {
		return big.NewInt(0)
	}

	return new(big.Int).Lsh(big.NewInt(1), uint(difficulty))
}

// ChainWork returns the total cumulative proof-of-work for a
// blockchain, summing 2^difficulty across every block.
func ChainWork(bc *Blockchain) *big.Int {

	total := big.NewInt(0)

	if bc == nil {
		return total
	}

	for _, b := range bc.Blocks {
		total.Add(total, blockWork(b.Difficulty))
	}

	return total
}


func (bc *Blockchain) ResolveFork(candidate *Blockchain) (bool, string) {

	if candidate == nil || len(candidate.Blocks) == 0 {
		return false, "candidate chain rejected: candidate is empty"
	}

	if len(bc.Blocks) == 0 {
		return false, "candidate chain rejected: current chain is empty (unexpected)"
	}

	if valid, msg := candidate.ValidateChain(); !valid {
		return false, fmt.Sprintf("candidate chain rejected: %s", msg)
	}

	if candidate.Blocks[0].Hash != bc.Blocks[0].Hash {
		return false, "candidate chain rejected: different genesis block"
	}

	currentWork := ChainWork(bc)
	candidateWork := ChainWork(candidate)

	cmp := candidateWork.Cmp(currentWork)

	accept := false
	tieBroken := false

	switch {

	case cmp > 0:
		accept = true

	case cmp == 0:
		// Deterministic tie-breaker: smaller final block hash wins.
		currentHead := bc.Blocks[len(bc.Blocks)-1].Hash
		candidateHead := candidate.Blocks[len(candidate.Blocks)-1].Hash

		if candidateHead < currentHead {
			accept = true
			tieBroken = true
		}
	}

	if !accept {

		return false, fmt.Sprintf(
			"candidate chain rejected: not more work and lost tie-break (candidate: %d blocks / work %s, current: %d blocks / work %s)",
			len(candidate.Blocks),
			candidateWork.String(),
			len(bc.Blocks),
			currentWork.String(),
		)
	}

	previousLength := len(bc.Blocks)
	previousWork := currentWork

	// Copy rather than alias candidate.Blocks, so later mutation of the
	// candidate object (e.g. by a caller reusing it) can't silently
	// corrupt this chain's blocks through a shared backing array.
	bc.Blocks = append([]block.Block(nil), candidate.Blocks...)

	reason := fmt.Sprintf(
		"candidate chain accepted: replaced %d-block chain (work %s) with %d-block chain (work %s)",
		previousLength,
		previousWork.String(),
		len(bc.Blocks),
		candidateWork.String(),
	)

	if tieBroken {
		reason += " [tie-break on final block hash]"
	}

	return true, reason
}