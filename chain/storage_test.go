package chain

import (
	"os"
	"testing"

	"toyblockchain/ledger"
)

func TestBlockchainSaveAndLoad(t *testing.T) {

	file := "test_blockchain.json"

	defer os.Remove(file)

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	)

	if err != nil {
		t.Fatal(err)
	}

	err = bc.SaveToFile(file)

	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFromFile(file)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Blocks) != len(bc.Blocks) {

		t.Error(
			"Loaded blockchain size mismatch",
		)
	}

	// Verify loaded chain is still valid.
	valid, msg := loaded.ValidateChain()

	if !valid {

		t.Errorf(
			"Loaded blockchain validation failed: %s",
			msg,
		)
	}
}

func TestPendingTransactionSaveLoad(t *testing.T) {

	file := "test_pending.json"

	defer os.Remove(file)

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		20,
	)

	txs := []ledger.Transaction{
		tx,
	}

	err := SavePending(
		file,
		txs,
	)

	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPending(file)

	if err != nil {
		t.Fatal(err)
	}

	if len(loaded) != 1 {

		t.Error(
			"Pending transaction loading failed",
		)
	}

	// Verify signature survived persistence.
	if !ledger.VerifyTransactionSignature(
		loaded[0],
	) {

		t.Error(
			"Loaded pending transaction has invalid signature",
		)
	}
}
