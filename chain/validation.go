package chain

import (
	"fmt"
	"strings"

	"toyblockchain/ledger"
)

// ValidateChain verifies the integrity of the entire blockchain.
func (bc *Blockchain) ValidateChain() (bool, string) {

	ld := ledger.NewLedger()

	for i := 0; i < len(bc.Blocks); i++ {

		current := bc.Blocks[i]

		//  Verify stored hash matches recalculated hash.
		if current.CalculateHash() != current.Hash {

			return false, fmt.Sprintf(
				"Block %d: hash mismatch (data tampered)",
				i,
			)
		}

		if i == 0 {

			// Validate genesis block.
			if current.Index != 0 {

				return false,
					"Genesis block has invalid index"
			}

			if current.PreviousHash != GenesisPreviousHash {

				return false,
					"Genesis block has invalid previous hash"
			}

		} else {

			previous := bc.Blocks[i-1]

			// Verify previous hash link.
			if current.PreviousHash != previous.Hash {

				return false, fmt.Sprintf(
					"Block %d: invalid previous hash link",
					i,
				)
			}

			//  Verify sequential block indexes.
			if current.Index != previous.Index+1 {

				return false, fmt.Sprintf(
					"Block %d: invalid block index",
					i,
				)
			}

			//  Verify timestamp ordering.
			if current.Timestamp < previous.Timestamp {

				return false, fmt.Sprintf(
					"Block %d: invalid timestamp",
					i,
				)
			}

			
			if current.Difficulty < MinDifficulty {

				return false, fmt.Sprintf(
					"Block %d: difficulty %d below minimum %d",
					i,
					current.Difficulty,
					MinDifficulty,
				)
			}

			//  Verify Proof-of-Work using the block's own difficulty.
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

		
		for _, tx := range current.Transactions {

			if err := ld.ApplyTransaction(tx); err != nil {

				return false, fmt.Sprintf(
					"Block %d: ledger replay failed: %v",
					i,
					err,
				)
			}
		}
	}

	return true, "Chain is valid"
}