package chain

import (
	"toyblockchain/crypto"
	"toyblockchain/ledger"
)

// createSignedTransaction builds a Transaction signed by a freshly
// generated key pair. Shared across chain package tests so the same
// helper logic isn't duplicated in multiple _test.go files.
func createSignedTransaction(
	sender string,
	receiver string,
	amount int64,
) ledger.Transaction {

	wallet, err := crypto.GenerateKeyPair()

	if err != nil {
		panic(err)
	}

	tx := ledger.Transaction{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
	}

	ledger.SignTransaction(&tx, wallet)

	return tx
}
