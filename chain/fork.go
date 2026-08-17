package chain

import "fmt"

func ChainWork(bc *Blockchain) int {

	work := 0

	for _, b := range bc.Blocks {
		work += b.Difficulty
	}

	return work
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

	currentLen := len(bc.Blocks)
	candidateLen := len(candidate.Blocks)

	accept := false

	switch {

	case candidateLen > currentLen:
		accept = true

	case candidateLen == currentLen:
		accept = ChainWork(candidate) > ChainWork(bc)

	}

	if !accept {

		return false, fmt.Sprintf(
			"candidate chain rejected: not longer and not more work (candidate: %d blocks / %d work, current: %d blocks / %d work)",
			candidateLen,
			ChainWork(candidate),
			currentLen,
			ChainWork(bc),
		)
	}

	previousLength := currentLen

	bc.Blocks = candidate.Blocks

	return true, fmt.Sprintf(
		"candidate chain accepted: replaced %d-block chain with %d-block chain",
		previousLength,
		candidateLen,
	)
}
