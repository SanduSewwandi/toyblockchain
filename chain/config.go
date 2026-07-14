package chain

import "runtime"

var (
	// Default mining difficulty.
	DefaultDifficulty = 4

	// Maximum number of transactions allowed in one block.
	DefaultBlockSize = 5

	// Blockchain persistence file.
	DefaultBlockchainFile = "blockchain.json"

	// Pending transaction persistence file.
	DefaultPendingFile = "pending.json"

	// Number of goroutines used for concurrent mining.
	DefaultMiningWorkers = runtime.NumCPU()
)

const MinDifficulty = 3
