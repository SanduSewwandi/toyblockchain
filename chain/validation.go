package chain

import (
	"fmt"
	"strings"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

// ValidateChain verifies the integrity of the entire blockchain.
func (bc *Blockchain) ValidateChain() (bool, string) {

	if bc == nil {
		return false, "Blockchain is nil"
	}

	if len(bc.Blocks) == 0 {
		return false, "Blockchain is empty"
	}

	ld := ledger.NewLedger()

	for i := 0; i < len(bc.Blocks); i++ {

		current := bc.Blocks[i]

		// Basic difficulty validation

		if current.Difficulty < MinDifficulty {

			return false, fmt.Sprintf(
				"Block %d: difficulty below minimum",
				i,
			)
		}

		// Verify Merkle root

		expectedMerkleRoot := block.MerkleRoot(
			current.Transactions,
		)

		if current.MerkleRoot != expectedMerkleRoot {

			return false, fmt.Sprintf(
				"Block %d: Merkle root mismatch (data tampered)",
				i,
			)
		}

		// Verify stored hash

		if current.CalculateHash() != current.Hash {

			return false, fmt.Sprintf(
				"Block %d: hash mismatch (data tampered)",
				i,
			)
		}

		// Genesis block validation

		if i == 0 {

			// Genesis must always have index 0.
			if current.Index != 0 {

				return false,
					"Genesis block has invalid index"
			}

			// Genesis must always point to the fixed zero hash.
			if current.PreviousHash != GenesisPreviousHash {

				return false,
					"Genesis block has invalid previous hash"
			}

		} else {

			previous := bc.Blocks[i-1]

			// Previous hash connection

			if current.PreviousHash != previous.Hash {

				return false, fmt.Sprintf(
					"Block %d: invalid previous hash link",
					i,
				)
			}

			// Block index

			if current.Index != previous.Index+1 {

				return false, fmt.Sprintf(
					"Block %d: invalid block index",
					i,
				)
			}

			// Timestamp
			if current.Timestamp < previous.Timestamp {

				return false, fmt.Sprintf(
					"Block %d: invalid timestamp",
					i,
				)
			}

			// Difficulty validation

			history := &Blockchain{
				Blocks: bc.Blocks[:i],
			}

			expectedDifficulty := NextDifficultyFor(
				history,
				DefaultDifficulty,
			)

			if current.Difficulty != expectedDifficulty {

				return false, fmt.Sprintf(
					"Block %d: invalid difficulty: expected %d, got %d",
					i,
					expectedDifficulty,
					current.Difficulty,
				)
			}

			// Proof of Work

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

		// Ledger validation

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
