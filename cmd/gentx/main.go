// Command gentx prints a signed ledger.Transaction as JSON, so it can be
// POSTed to a running node's /transactions endpoint for manual testing.
//
// Usage:
//
//	go run ./cmd/gentx -sender Alice -receiver Bob -amount 10
//
// Each run generates a FRESH key pair for the sender (it does not read
// wallet.json), so re-running with the same -sender name will NOT reuse
// the same key. That's fine for a first transaction from a brand-new
// name, but if you want to send a SECOND transaction from the same
// "Alice", save the printed public/private key from the first run and
// wire in a -reuse-key flag, or just use a different sender name each
// time for quick manual testing.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"toyblockchain/crypto"
	"toyblockchain/ledger"
	"toyblockchain/wallet"
)

func main() {
	sender := flag.String("sender", "Alice", "sender name")
	receiver := flag.String("receiver", "Bob", "receiver name")
	amount := flag.Int64("amount", 10, "amount to send")
	keyfile := flag.String("keyfile", "", "wallet file to load/persist a reusable key for -sender (omit for a fresh random key each run)")
	flag.Parse()

	var kp crypto.KeyPair
	var err error

	if *keyfile != "" {
		w, werr := wallet.LoadWallet(*keyfile)
		if werr != nil {
			fmt.Fprintln(os.Stderr, "failed to load wallet:", werr)
			os.Exit(1)
		}

		kp, err = w.GetOrCreate(*sender)
	} else {
		kp, err = crypto.GenerateKeyPair()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to obtain key pair:", err)
		os.Exit(1)
	}

	tx := ledger.Transaction{
		Sender:   *sender,
		Receiver: *receiver,
		Amount:   *amount,
	}

	ledger.SignTransaction(&tx, kp)

	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to marshal transaction:", err)
		os.Exit(1)
	}

	fmt.Println(string(data))
}
