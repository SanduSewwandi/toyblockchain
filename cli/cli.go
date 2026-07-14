package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"toyblockchain/chain"
	"toyblockchain/ledger"
	"toyblockchain/wallet"
)

func isValidSender(sender string) bool {

	return sender != "" && !strings.EqualFold(sender, "SYSTEM")
}

func Run() {

	run(flag.Args())
}

func run(args []string) {

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

	// Load wallet (named accounts and their key pairs). Created fresh
	// with no accounts if this is the first run.
	w, err := wallet.LoadWallet(wallet.DefaultWalletFile)

	if err != nil {

		fmt.Println(
			"Error loading wallet:",
			err,
		)

		return
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

		if !isValidSender(args[1]) {

			fmt.Println(
				"Invalid sender: cannot mint funds from an empty or reserved sender",
			)

			return
		}

		amount, err := strconv.ParseInt(
			args[3],
			10,
			64,
		)

		if err != nil {

			fmt.Println(
				"Invalid amount",
			)

			return
		}

		// Look up (or create, on first use) the sender's key pair, and
		// sign the transaction with it.
		senderKeys, err := w.GetOrCreate(args[1])

		if err != nil {

			fmt.Println(
				"Error accessing sender's key pair:",
				err,
			)

			return
		}

		tx := ledger.Transaction{

			Sender: args[1],

			Receiver: args[2],

			Amount: amount,
		}

		ledger.SignTransaction(&tx, senderKeys)

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

		toMine := pendingTransactions
		remaining := []ledger.Transaction{}

		if len(pendingTransactions) > chain.DefaultBlockSize {

			toMine = pendingTransactions[:chain.DefaultBlockSize]
			remaining = pendingTransactions[chain.DefaultBlockSize:]
		}

		tempLedger := ld.Clone()

		for _, pending := range toMine {

			if !isValidSender(pending.Sender) {

				fmt.Println(
					"Pending pool contains a transaction with an invalid sender, aborting mine:",
					pending,
				)

				return
			}

			if err := tempLedger.ApplyTransaction(pending); err != nil {

				fmt.Println(
					"Pending pool contains an invalid transaction, aborting mine:",
					err,
				)

				return
			}
		}

		fmt.Println(
			"Mining difficulty:",
			chain.DefaultDifficulty,
		)

		err := blockchain.AddBlock(
			toMine,
			chain.DefaultDifficulty,
		)

		if err != nil {

			fmt.Println(
				"Mining failed:",
				err,
			)

			return
		}

		// AddBlock no longer saves as a side effect — save explicitly.
		if err := blockchain.SaveToFile(chain.DefaultBlockchainFile); err != nil {

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

		if err := chain.SavePending(
			chain.DefaultPendingFile,
			remaining,
		); err != nil {

			fmt.Println(
				"Error saving pending transactions:",
				err,
			)

			return
		}

		if len(remaining) > 0 {

			fmt.Printf(
				"%d transaction(s) remain pending for the next block.\n",
				len(remaining),
			)
		}

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

		aliceKeys, err := w.GetOrCreate("Alice")

		if err != nil {

			fmt.Println(
				"Error accessing Alice's key pair:",
				err,
			)

			return
		}

		bobKeys, err := w.GetOrCreate("Bob")

		if err != nil {

			fmt.Println(
				"Error accessing Bob's key pair:",
				err,
			)

			return
		}

		tx1 := ledger.Transaction{
			Sender:   "Alice",
			Receiver: "Bob",
			Amount:   20,
		}

		ledger.SignTransaction(&tx1, aliceKeys)

		tx2 := ledger.Transaction{
			Sender:   "Bob",
			Receiver: "Charlie",
			Amount:   10,
		}

		ledger.SignTransaction(&tx2, bobKeys)

		if err := blockchain.AddBlock(
			[]ledger.Transaction{tx1},
			chain.DefaultDifficulty,
		); err != nil {

			fmt.Println(
				"Error adding first block:",
				err,
			)

			return
		}

		if err := blockchain.AddBlock(
			[]ledger.Transaction{tx2},
			chain.DefaultDifficulty,
		); err != nil {

			fmt.Println(
				"Error adding second block:",
				err,
			)

			return
		}

		// Save blockchain explicitly.
		if err := blockchain.SaveToFile(
			chain.DefaultBlockchainFile,
		); err != nil {

			fmt.Println(
				"Error saving blockchain:",
				err,
			)

			return
		}

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
