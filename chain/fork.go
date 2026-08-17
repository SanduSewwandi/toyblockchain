package chain

import (
	"fmt"
	"math/big"
)

// blockWork returns the expected number of hashes needed to mine a block
// at the given difficulty. Difficulty here counts required leading hex
// characters in the hash, and each hex character encodes 4 bits, so the
// expected search cost scales as 16^difficulty, not difficulty itself.
func blockWork(difficulty int) *big.Int {

	work := big.NewInt(1)
	sixteen := big.NewInt(16)

	for i := 0; i < difficulty; i++ {
		work.Mul(work, sixteen)
	}

	return work
}

// ChainWork returns the total cumulative proof-of-work a chain
// represents: the sum of the expected mining cost of every block in it.
func ChainWork(bc *Blockchain) *big.Int {

	total := big.NewInt(0)

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

	candidateWork := ChainWork(candidate)
	currentWork := ChainWork(bc)

	accept := false

	switch candidateWork.Cmp(currentWork) {

	case 1:
		// Candidate has strictly more cumulative proof-of-work — wins
		// regardless of block count.
		accept = true

	case 0:
		// Equal work: fall back to a deterministic tie-breaker (final
		// block hash) so every node in the network converges on the
		// same winner, whichever side runs the comparison.
		accept = candidate.Blocks[len(candidate.Blocks)-1].Hash >
			bc.Blocks[len(bc.Blocks)-1].Hash
	}

	if !accept {

		return false, fmt.Sprintf(
			"candidate chain rejected: not more cumulative work (candidate: %d blocks / work %s, current: %d blocks / work %s)",
			len(candidate.Blocks),
			candidateWork.String(),
			len(bc.Blocks),
			currentWork.String(),
		)
	}

	previousLength := len(bc.Blocks)
	previousWork := currentWork.String()
	candidateLength := len(candidate.Blocks)

	bc.Blocks = candidate.Blocks

	return true, fmt.Sprintf(
		"candidate chain accepted: replaced %d-block chain (work %s) with %d-block chain (work %s)",
		previousLength,
		previousWork,
		candidateLength,
		candidateWork.String(),
	)
}
