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

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

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

}

func TestPendingTransactionSaveLoad(t *testing.T) {

	file := "test_pending.json"

	defer os.Remove(file)

	txs := []ledger.Transaction{

		{
			Sender:   "Alice",
			Receiver: "Bob",
			Amount:   20,
		},
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

}
