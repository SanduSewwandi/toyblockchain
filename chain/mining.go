package chain

import (
	"fmt"
	"strings"
	"time"

	"toyblockchain/block"
)

func MineBlock(b *block.Block, difficulty int) {

	// Store the difficulty in the block.
	b.Difficulty = difficulty

	// Calculate Merkle root before mining.

	b.MerkleRoot = block.MerkleRoot(
		b.Transactions,
	)

	// Target prefix (example: "0000")
	target := strings.Repeat(
		"0",
		difficulty,
	)

	fmt.Println("\n===================================")
	fmt.Println("Mining Block...")
	fmt.Printf(
		"Difficulty : %d leading zeros\n",
		difficulty,
	)

	start := time.Now()

	for {

		// Calculate current hash.
		hash := b.CalculateHash()

		// Check difficulty requirement.
		if strings.HasPrefix(
			hash,
			target,
		) {

			b.Hash = hash
			break
		}

		// Try the next nonce.
		b.Nonce++
	}

	elapsed := time.Since(start)

	fmt.Println("Mining Successful!")
	fmt.Printf(
		"Nonce        : %d\n",
		b.Nonce,
	)

	fmt.Printf(
		"Merkle Root  : %s\n",
		b.MerkleRoot,
	)

	fmt.Printf(
		"Hash         : %s\n",
		b.Hash,
	)

	fmt.Printf(
		"Mining Time  : %v\n",
		elapsed,
	)

	fmt.Println("===================================")
}
