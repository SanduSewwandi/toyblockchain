package cli

import (
	"fmt"
	"os"
	"strconv"

	"toyblockchain/chain"
	"toyblockchain/ledger"
)

func Run() {

	// Load blockchain from disk
	blockchain, err := chain.LoadFromFile(chain.DefaultBlockchainFile)
	if err != nil {
		fmt.Println("Error loading blockchain:", err)
		return
	}

	// Build ledger from blockchain
	ld := blockchain.BuildLedger()

	// Load pending transactions
	pendingTransactions, err := chain.LoadPending(chain.DefaultPendingFile)
	if err != nil {
		fmt.Println("Error loading pending transactions:", err)
		return
	}

	args := os.Args

	if len(args) < 2 {
		printHelp()
		return
	}

	switch args[1] {

	case "add":

		if len(args) != 5 {

			fmt.Println("Usage:")
			fmt.Println(
				"go run main.go add <sender> <receiver> <amount>",
			)

			return
		}

		amount, err := strconv.ParseFloat(args[4], 64)

		if err != nil {

			fmt.Println("Invalid amount.")
			return
		}

		tx := ledger.Transaction{

			Sender: args[2],

			Receiver: args[3],

			Amount: amount,
		}

		// Validate transaction with pending pool

		// Create temporary ledger copy
		tempLedger := ld.Clone()

		// Apply all existing pending transactions first
		for _, pendingTx := range pendingTransactions {

			err := tempLedger.ApplyTransaction(pendingTx)

			if err != nil {

				fmt.Println(
					"Invalid pending transaction:",
					err,
				)

				return
			}
		}

		// Test the new transaction
		if err := tempLedger.ApplyTransaction(tx); err != nil {

			fmt.Println(
				"Transaction rejected:",
				err,
			)

			return
		}

		// Add to pending pool
		pendingTransactions = append(
			pendingTransactions,
			tx,
		)

		// Save pending transactions
		if err := chain.SavePending(
			chain.DefaultPendingFile,
			pendingTransactions,
		); err != nil {

			fmt.Println(
				"Error saving pending transaction:",
				err,
			)

			return
		}

		fmt.Println(
			"Transaction added to pending pool.",
		)

		fmt.Printf(
			"Pending transactions: %d\n",
			len(pendingTransactions),
		)

	case "mine":

		if len(pendingTransactions) == 0 {

			fmt.Println(
				"No pending transactions to mine.",
			)

			return
		}

		// Default difficulty
		difficulty := chain.DefaultDifficulty

		// Custom difficulty
		if len(args) >= 3 {

			value, err := strconv.Atoi(args[2])

			if err != nil {

				fmt.Println(
					"Invalid difficulty.",
				)

				return
			}

			difficulty = value
		}

		fmt.Println(
			"Mining difficulty:",
			difficulty,
		)

		// Mine block
		if err := blockchain.AddBlock(
			pendingTransactions,
			difficulty,
		); err != nil {

			fmt.Println(
				"Failed to mine block:",
				err,
			)

			return
		}

		fmt.Println(
			"Block mined successfully.",
		)

		fmt.Println(
			"Blockchain saved to",
			chain.DefaultBlockchainFile,
		)

		// Clear pending transactions
		if err := chain.ClearPending(
			chain.DefaultPendingFile,
		); err != nil {

			fmt.Println(
				"Error clearing pending transactions:",
				err,
			)

			return
		}

	case "print":

		blockchain.Print()

	case "validate":

		valid, message := blockchain.ValidateChain()

		fmt.Println(
			"========== VALIDATION ==========",
		)

		fmt.Println(
			"Valid  :",
			valid,
		)

		fmt.Println(
			"Message:",
			message,
		)

	case "balance":

		ld.Print()

	case "demo":

		tx1 := ledger.Transaction{

			Sender: "Alice",

			Receiver: "Bob",

			Amount: 20,
		}

		tx2 := ledger.Transaction{

			Sender: "Bob",

			Receiver: "Charlie",

			Amount: 10,
		}

		if err := ld.ApplyTransaction(tx1); err != nil {

			fmt.Println(err)

			return
		}

		if err := ld.ApplyTransaction(tx2); err != nil {

			fmt.Println(err)

			return
		}

		// Mine first block
		if err := blockchain.AddBlock(
			[]ledger.Transaction{tx1},
			chain.DefaultDifficulty,
		); err != nil {

			fmt.Println(err)

			return
		}

		// Mine second block
		if err := blockchain.AddBlock(
			[]ledger.Transaction{tx2},
			chain.DefaultDifficulty,
		); err != nil {

			fmt.Println(err)

			return
		}

		// Rebuild ledger
		ld = blockchain.BuildLedger()

		blockchain.Print()

		ld.Print()

		valid, message := blockchain.ValidateChain()

		fmt.Println(
			"\n========== VALIDATION (BEFORE TAMPERING) ==========",
		)

		fmt.Println(
			"Valid:",
			valid,
		)

		fmt.Println(
			"Message:",
			message,
		)

		fmt.Println(
			"\nTampering with Block 1 transaction...",
		)

		blockchain.Blocks[1].
			Transactions[0].
			Amount = 9999

		valid, message = blockchain.ValidateChain()

		fmt.Println(
			"\n========== VALIDATION (AFTER TAMPERING) ==========",
		)

		fmt.Println(
			"Valid:",
			valid,
		)

		fmt.Println(
			"Message:",
			message,
		)

	default:

		fmt.Println(
			"Unknown command.",
		)

		printHelp()
	}
}

// printHelp displays available commands.
func printHelp() {

	fmt.Println(
		"===================================",
	)

	fmt.Println(
		"Toy Blockchain CLI",
	)

	fmt.Println(
		"===================================",
	)

	fmt.Println()

	fmt.Println("Commands:")

	fmt.Println()

	fmt.Println("  demo")

	fmt.Println(
		"      Run complete demonstration",
	)

	fmt.Println()

	fmt.Println(
		"  add <sender> <receiver> <amount>",
	)

	fmt.Println(
		"      Add transaction to pending pool",
	)

	fmt.Println()

	fmt.Println(
		"  mine [difficulty]",
	)

	fmt.Println(
		"      Mine pending transactions",
	)

	fmt.Println(
		"      Example: go run main.go mine 5",
	)

	fmt.Println()

	fmt.Println(
		"  print",
	)

	fmt.Println(
		"      Display blockchain",
	)

	fmt.Println()

	fmt.Println(
		"  validate",
	)

	fmt.Println(
		"      Validate blockchain integrity",
	)

	fmt.Println()

	fmt.Println(
		"  balance",
	)

	fmt.Println(
		"      Display ledger balances",
	)
}
