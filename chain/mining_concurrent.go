package chain

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"toyblockchain/block"
)

func MineBlockConcurrent(b *block.Block, difficulty int, numWorkers int) {

	if numWorkers < 1 {
		numWorkers = 1
	}

	b.Difficulty = difficulty

	// Ensure the Merkle root reflects the block's current transactions
	// before any worker starts hashing.
	b.MerkleRoot = block.MerkleRoot(b.Transactions)

	target := strings.Repeat("0", difficulty)

	var found atomic.Bool
	var winner atomic.Value // holds a block.Block once set

	var wg sync.WaitGroup

	fmt.Println("\n===================================")
	fmt.Println("Mining Block (concurrent)...")
	fmt.Printf("Difficulty : %d leading zeros\n", difficulty)
	fmt.Printf("Workers    : %d\n", numWorkers)

	start := time.Now()

	for w := 0; w < numWorkers; w++ {

		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			local := *b
			nonce := workerID

			for !found.Load() {

				local.Nonce = nonce
				hash := local.CalculateHash()

				if strings.HasPrefix(hash, target) {

					if found.CompareAndSwap(false, true) {
						local.Hash = hash
						winner.Store(local)
					}

					return
				}

				nonce += numWorkers
			}
		}(w)
	}

	wg.Wait()

	elapsed := time.Since(start)

	result := winner.Load().(block.Block)
	b.Nonce = result.Nonce
	b.Hash = result.Hash

	fmt.Println("Mining Successful!")
	fmt.Printf("Nonce        : %d\n", b.Nonce)
	fmt.Printf("Merkle Root  : %s\n", b.MerkleRoot)
	fmt.Printf("Hash         : %s\n", b.Hash)
	fmt.Printf("Mining Time  : %v\n", elapsed)
	fmt.Println("===================================")
}
