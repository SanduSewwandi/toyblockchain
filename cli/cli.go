package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"toyblockchain/chain"
	"toyblockchain/ledger"
)

func Run() {

	args := flag.Args()

	if len(args) < 1 {
		printHelp()
		return
	}

	// Load blockchain
	blockchain, err := chain.LoadFromFile(
		chain.DefaultBlockchainFile,
	)

	if err != nil {

		if os.IsNotExist(err) {

			fmt.Println(
				"Blockchain file not found. Creating new blockchain...",
			)

			blockchain = chain.NewBlockchain()

			// Save new blockchain
			if err := blockchain.SaveToFile(
				chain.DefaultBlockchainFile,
			); err != nil {

				fmt.Println(
					"Error saving blockchain:",
					err,
				)

				return
			}

		} else {

			fmt.Println(
				"Error loading blockchain:",
				err,
			)

			return
		}
	}

	// Build ledger
	ld := blockchain.BuildLedger()

	// Load pending transactions
	pendingTransactions, err := chain.LoadPending(
		chain.DefaultPendingFile,
	)

	if err != nil {

		fmt.Println(
			"Error loading pending transactions:",
			err,
		)

		return
	}

	switch args[0] {

	case "add":

		if len(args) != 4 {

			fmt.Println(
				"Usage: go run main.go add <sender> <receiver> <amount>",
			)

			return
		}

		amount, err := strconv.ParseFloat(
			args[3],
			64,
		)

		if err != nil {

			fmt.Println(
				"Invalid amount",
			)

			return
		}

		tx := ledger.Transaction{

			Sender: args[1],

			Receiver: args[2],

			Amount: amount,
		}

		tempLedger := ld.Clone()

		// Apply pending transactions first
		for _, pending := range pendingTransactions {

			if err := tempLedger.ApplyTransaction(
				pending,
			); err != nil {

				fmt.Println(
					"Invalid pending transaction:",
					err,
				)

				return
			}
		}

		// Validate new transaction

		if err := tempLedger.ApplyTransaction(
			tx,
		); err != nil {

			fmt.Println(
				"Transaction rejected:",
				err,
			)

			return
		}

		pendingTransactions = append(
			pendingTransactions,
			tx,
		)

		if err := chain.SavePending(
			chain.DefaultPendingFile,
			pendingTransactions,
		); err != nil {

			fmt.Println(
				"Error saving pending transactions:",
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

		if len(pendingTransactions) >
			chain.DefaultBlockSize {

			fmt.Printf(
				"Too many transactions. Maximum block size is %d\n",
				chain.DefaultBlockSize,
			)

			return
		}

		fmt.Println(
			"Mining difficulty:",
			chain.DefaultDifficulty,
		)

		err := blockchain.AddBlock(
			pendingTransactions,
			chain.DefaultDifficulty,
		)

		if err != nil {

			fmt.Println(
				"Mining failed:",
				err,
			)

			return
		}

		// Save blockchain

		err = blockchain.SaveToFile(
			chain.DefaultBlockchainFile,
		)

		if err != nil {

			fmt.Println(
				"Error saving blockchain:",
				err,
			)

			return
		}

		fmt.Println(
			"Block mined successfully.",
		)

		fmt.Println(
			"Saved file:",
			chain.DefaultBlockchainFile,
		)

		chain.ClearPending(
			chain.DefaultPendingFile,
		)

	case "print":

		blockchain.Print()

	case "validate":

		valid, msg := blockchain.ValidateChain()

		fmt.Println(
			"========== VALIDATION ==========",
		)

		fmt.Println(
			"Valid:",
			valid,
		)

		fmt.Println(
			"Message:",
			msg,
		)

	case "balance":

		ld.Print()

	case "demo":

		fmt.Println(
			"Running blockchain demo...",
		)

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

		blockchain.AddBlock(
			[]ledger.Transaction{tx1},
			chain.DefaultDifficulty,
		)

		blockchain.AddBlock(
			[]ledger.Transaction{tx2},
			chain.DefaultDifficulty,
		)

		blockchain.Print()

		valid, msg := blockchain.ValidateChain()

		fmt.Println(
			"Valid:",
			valid,
		)

		fmt.Println(
			msg,
		)

	case "help":

		printHelp()

	default:

		fmt.Println(
			"Unknown command",
		)

		printHelp()

	}

}

func printHelp() {

	fmt.Println("===============================")

	fmt.Println(
		"Toy Blockchain CLI",
	)

	fmt.Println("===============================")

	fmt.Println()

	fmt.Println("Flags:")

	fmt.Println(
		" -difficulty=N   Mining difficulty",
	)

	fmt.Println(
		" -blocksize=N    Maximum transactions/block",
	)

	fmt.Println(
		" -data=file.json Blockchain storage file",
	)

	fmt.Println()

	fmt.Println("Commands:")

	fmt.Println(
		" add <sender> <receiver> <amount>",
	)

	fmt.Println(
		" mine",
	)

	fmt.Println(
		" print",
	)

	fmt.Println(
		" validate",
	)

	fmt.Println(
		" balance",
	)

	fmt.Println(
		" demo",
	)

}
