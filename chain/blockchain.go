package chain

import (
	"fmt"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

// GenesisPreviousHash is the fixed previous hash used by the genesis block.
const GenesisPreviousHash = "0000000000000000000000000000000000000000000000000000000000000000"

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

	genesis := block.NewBlock(
		0,
		genesisTransactions,
		GenesisPreviousHash,
		DefaultDifficulty,
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

// AddBlock mines a new block and appends it to the chain.
func (bc *Blockchain) AddBlock(
	transactions []ledger.Transaction,
	difficulty int,
) error {

	// Enforce configurable maximum block size (FR-9).
	if len(transactions) > DefaultBlockSize {

		return fmt.Errorf(
			"block contains %d transactions, maximum allowed is %d",
			len(transactions),
			DefaultBlockSize,
		)
	}

	latest := bc.GetLatestBlock()

	newBlock := block.NewBlock(
		latest.Index+1,
		transactions,
		latest.Hash,
		difficulty,
	)

	// Mine block using selected difficulty.
	MineBlock(
		&newBlock,
		difficulty,
	)

	// Append mined block.
	bc.Blocks = append(
		bc.Blocks,
		newBlock,
	)

	// Save blockchain.
	if err := bc.SaveToFile(DefaultBlockchainFile); err != nil {
		return err
	}

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
					"  %d. %s -> %s : %.2f\n",
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

// BuildLedger reconstructs balances only from blockchain transactions.
//
// No balances are stored separately.
// The ledger is created by replaying every transaction in every block.
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
