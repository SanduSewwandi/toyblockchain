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
				return false, "Genesis block has invalid index"
			}

			if current.PreviousHash != GenesisPreviousHash {
				return false, "Genesis block has invalid previous hash"
			}

		} else {

			previous := bc.Blocks[i-1]

			// Previous hash connection check
			if current.PreviousHash != previous.Hash {

				return false, fmt.Sprintf(
					"Block %d: invalid previous hash link",
					i,
				)
			}

			// Block index check
			if current.Index != previous.Index+1 {

				return false, fmt.Sprintf(
					"Block %d: invalid block index",
					i,
				)
			}

			// Timestamp check
			if current.Timestamp < previous.Timestamp {

				return false, fmt.Sprintf(
					"Block %d: invalid timestamp",
					i,
				)
			}

			// Difficulty check
			if current.Difficulty < MinDifficulty {

				return false, fmt.Sprintf(
					"Block %d: difficulty below minimum",
					i,
				)
			}

			// Proof of Work check
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

		// Verify transactions. Signature validity and identity binding
		// (matching each sender to their registered public key) are
		// both checked inside ApplyTransaction, so a single call here
		// covers everything: signature, identity, and balance.
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
