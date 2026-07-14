package chain

import (
	"fmt"
	"strings"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

// ValidateChain verifies the integrity of the entire blockchain.
func (bc *Blockchain) ValidateChain() (bool, string) {

	if len(bc.Blocks) == 0 {

		return false, "Blockchain is empty"
	}

	ld := ledger.NewLedger()

	for i := 0; i < len(bc.Blocks); i++ {

		current := bc.Blocks[i]

		// Verify Merkle root matches transactions.
		expectedMerkleRoot := block.MerkleRoot(
			current.Transactions,
		)

		if current.MerkleRoot != expectedMerkleRoot {

			return false, fmt.Sprintf(
				"Block %d: Merkle root mismatch (data tampered)",
				i,
			)
		}

		// Verify stored hash matches recalculated hash.
		if current.CalculateHash() != current.Hash {

			return false, fmt.Sprintf(
				"Block %d: hash mismatch (data tampered)",
				i,
			)
		}

		// Genesis block validation
		if i == 0 {

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

			// Verify block index sequence.
			if current.Index != previous.Index+1 {

				return false, fmt.Sprintf(
					"Block %d: invalid block index",
					i,
				)
			}

			// Verify timestamp ordering.
			if current.Timestamp < previous.Timestamp {

				return false, fmt.Sprintf(
					"Block %d: invalid timestamp",
					i,
				)
			}

			// Verify difficulty value.
			if current.Difficulty < MinDifficulty {

				return false, fmt.Sprintf(
					"Block %d: difficulty %d below minimum %d",
					i,
					current.Difficulty,
					MinDifficulty,
				)
			}

			// Verify Proof-of-Work.
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

		// Replay transactions to verify ledger consistency.
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
