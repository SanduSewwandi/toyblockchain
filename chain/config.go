package chain

var (
	// Default mining difficulty.
	DefaultDifficulty = 4

	// Maximum number of transactions allowed in one block.
	DefaultBlockSize = 5

	// Blockchain persistence file.
	DefaultBlockchainFile = "blockchain.json"

	// Pending transaction persistence file.
	DefaultPendingFile = "pending.json"
)

const MinDifficulty = 3
