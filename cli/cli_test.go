package cli

import (
	"path/filepath"
	"testing"

	"toyblockchain/chain"
	"toyblockchain/ledger"
)

func TestIsValidSender(t *testing.T) {

	cases := map[string]bool{
		"Alice":  true,
		"Bob":    true,
		"":       false,
		"SYSTEM": false,
		"system": false,
		"System": false,
	}

	for sender, want := range cases {

		if got := isValidSender(sender); got != want {

			t.Errorf(
				"isValidSender(%q) = %v, want %v",
				sender,
				got,
				want,
			)
		}
	}
}

func TestMineRejectsHandEditedOverspend(t *testing.T) {

	dir := t.TempDir()

	dataFile := filepath.Join(dir, "blockchain.json")
	pendingFile := filepath.Join(dir, "pending.json")

	// Point the package-level config at isolated temp files so this test
	// can't interfere with other tests or leave artifacts behind.
	oldData, oldPending := chain.DefaultBlockchainFile, chain.DefaultPendingFile
	chain.DefaultBlockchainFile = dataFile
	chain.DefaultPendingFile = pendingFile
	defer func() {
		chain.DefaultBlockchainFile = oldData
		chain.DefaultPendingFile = oldPending
	}()

	// Fresh chain: Alice starts with 100 from the genesis coinbase tx.
	bc := chain.NewBlockchain()
	if err := bc.SaveToFile(dataFile); err != nil {
		t.Fatal(err)
	}

	badPending := []ledger.Transaction{
		{Sender: "Alice", Receiver: "Mallory", Amount: 999999},
	}
	if err := chain.SavePending(pendingFile, badPending); err != nil {
		t.Fatal(err)
	}

	// Run the "mine" command against the tampered pending pool.
	run([]string{"mine"})

	loaded, err := chain.LoadFromFile(dataFile)
	if err != nil {
		t.Fatalf("chain failed to reload: %v", err)
	}

	if len(loaded.Blocks) != 1 {
		t.Errorf(
			"expected mine to abort and leave the chain at genesis only, got %d blocks",
			len(loaded.Blocks),
		)
	}

	stillPending, err := chain.LoadPending(pendingFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(stillPending) != 1 {
		t.Errorf(
			"expected the rejected transaction to remain pending, got %d pending",
			len(stillPending),
		)
	}
}
