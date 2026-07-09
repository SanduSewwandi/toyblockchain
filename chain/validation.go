package chain

import (
	"fmt"
	"strings"
)

// ValidateChain verifies the integrity of the entire blockchain.
func (bc *Blockchain) ValidateChain() (bool, string) {

	for i := 0; i < len(bc.Blocks); i++ {

		current := bc.Blocks[i]

		// 1. Verify stored hash matches recalculated hash.
		if current.CalculateHash() != current.Hash {

			return false, fmt.Sprintf(
				"Block %d: hash mismatch (data tampered)",
				i,
			)
		}

		// 2. Validate genesis block.
		if i == 0 {

			if current.Index != 0 {

				return false,
					"Genesis block has invalid index"
			}

			if current.PreviousHash != GenesisPreviousHash {

				return false,
					"Genesis block has invalid previous hash"
			}

			// Genesis block does not have a previous block.
			continue
		}

		previous := bc.Blocks[i-1]

		// 3. Verify previous hash link.
		if current.PreviousHash != previous.Hash {

			return false, fmt.Sprintf(
				"Block %d: invalid previous hash link",
				i,
			)
		}

		// 4. Verify sequential block indexes.
		if current.Index != previous.Index+1 {

			return false, fmt.Sprintf(
				"Block %d: invalid block index",
				i,
			)
		}

		// 5. Verify timestamp ordering.
		if current.Timestamp < previous.Timestamp {

			return false, fmt.Sprintf(
				"Block %d: invalid timestamp",
				i,
			)
		}

		// 6. Verify stored difficulty.
		if current.Difficulty <= 0 {

			return false, fmt.Sprintf(
				"Block %d: invalid difficulty",
				i,
			)
		}

		// 7. Verify Proof-of-Work using the block's own difficulty.
		target := strings.Repeat(
			"0",
			current.Difficulty,
		)

		if !strings.HasPrefix(
			current.Hash,
			target,
		) {

			return false, fmt.Sprintf(
				"Block %d: invalid proof-of-work",
				i,
			)
		}
	}

	return true, "Chain is valid"
}
