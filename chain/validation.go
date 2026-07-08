package chain

import (
	"fmt"
	"strings"
)

func (bc *Blockchain) ValidateChain() (bool, string) {

	// Use default mining difficulty
	target := strings.Repeat("0", DefaultDifficulty)

	for i := 0; i < len(bc.Blocks); i++ {

		current := bc.Blocks[i]

		if current.CalculateHash() != current.Hash {

			return false, fmt.Sprintf(
				"Block %d: hash mismatch (data tampered)",
				i,
			)
		}

		if i == 0 {

			if current.Index != 0 {

				return false,
					"Genesis block has invalid index"
			}

			continue
		}

		previous := bc.Blocks[i-1]

		if current.PreviousHash != previous.Hash {

			return false, fmt.Sprintf(
				"Block %d: invalid previous hash link",
				i,
			)
		}

		if current.Index != previous.Index+1 {

			return false, fmt.Sprintf(
				"Block %d: invalid block index",
				i,
			)
		}

		// 5. Verify timestamp order
		if current.Timestamp < previous.Timestamp {

			return false, fmt.Sprintf(
				"Block %d: invalid timestamp",
				i,
			)
		}

		// 6. Verify Proof-of-Work
		if !strings.HasPrefix(current.Hash, target) {

			return false, fmt.Sprintf(
				"Block %d: invalid proof-of-work",
				i,
			)
		}
	}

	return true, "Chain is valid"
}
