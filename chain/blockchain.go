package chain

import (
	"fmt"
	"strings"
	"toyblockchain/block"
	"toyblockchain/ledger"
)

var GenesisPreviousHash = strings.Repeat("0", 64)

const GenesisTimestamp int64 = 0

type Blockchain struct {
	Blocks []block.Block
}

func NewBlockchain() *Blockchain {

	genesisTransactions := []ledger.Transaction{

		{
			// Empty sender represents system/faucet creation.
			Sender:   "",
			Receiver: "Alice",
			Amount:   100,
		},

		{
			Sender:   "",
			Receiver: "Bob",
			Amount:   50,
		},
	}

	genesis := block.NewBlockWithTimestamp(
		0,
		genesisTransactions,
		GenesisPreviousHash,
		DefaultDifficulty,
		GenesisTimestamp,
	)

	return &Blockchain{
		Blocks: []block.Block{genesis},
	}
}

// GetLatestBlock returns the newest block.
func (bc *Blockchain) GetLatestBlock() block.Block {

	if len(bc.Blocks) == 0 {
		panic("blockchain is empty")
	}

	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) AddBlock(
	transactions []ledger.Transaction,
	difficulty int,
) error {

	// Enforce maximum block size.
	if len(transactions) > DefaultBlockSize {

		return fmt.Errorf(
			"block contains %d transactions, maximum allowed is %d",
			len(transactions),
			DefaultBlockSize,
		)
	}

	// Validate transactions against replayed ledger state before mining,
	// so an overspend or invalid signature can never be mined into a
	// block regardless of which caller invoked AddBlock.
	ld := bc.BuildLedger()

	for _, tx := range transactions {

		if err := ld.ApplyTransaction(tx); err != nil {

			return fmt.Errorf(
				"invalid transaction in block: %w",
				err,
			)
		}
	}

	nextDifficulty := NextDifficultyFor(bc, difficulty)

	latest := bc.GetLatestBlock()

	newBlock := block.NewBlock(
		latest.Index+1,
		transactions,
		latest.Hash,
		nextDifficulty,
	)

	MineBlockConcurrent(
		&newBlock,
		nextDifficulty,
		DefaultMiningWorkers,
	)

	// Append mined block.
	bc.Blocks = append(
		bc.Blocks,
		newBlock,
	)

	return nil
}

// Print displays the blockchain.
func (bc *Blockchain) Print() {

	fmt.Println("\n========== BLOCKCHAIN ==========")

	for _, b := range bc.Blocks {

		fmt.Println("--------------------------------")
		fmt.Printf("Index         : %d\n", b.Index)
		fmt.Printf("Timestamp     : %d\n", b.Timestamp)
		fmt.Printf("Previous Hash : %s\n", b.PreviousHash)
		fmt.Printf("Merkle Root   : %s\n", b.MerkleRoot)
		fmt.Printf("Hash          : %s\n", b.Hash)
		fmt.Printf("Nonce         : %d\n", b.Nonce)
		fmt.Printf("Difficulty    : %d\n", b.Difficulty)

		fmt.Println("Transactions:")

		if len(b.Transactions) == 0 {

			fmt.Println("  No transactions")

		} else {

			for i, tx := range b.Transactions {

				sender := tx.Sender

				if sender == "" {
					sender = "SYSTEM"
				}

				fmt.Printf(
					"  %d. %s -> %s : %d\n",
					i+1,
					sender,
					tx.Receiver,
					tx.Amount,
				)
			}
		}

		fmt.Println("--------------------------------")
	}
}

func (bc *Blockchain) BuildLedger() *ledger.Ledger {

	ld := ledger.NewLedger()

	// Replay all transactions from blockchain.
	for _, b := range bc.Blocks {

		for _, tx := range b.Transactions {

			if err := ld.ApplyTransaction(tx); err != nil {

				fmt.Println(
					"Warning:",
					err,
				)
			}
		}
	}

	return ld
}