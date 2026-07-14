package block

import (
	"testing"

	"toyblockchain/ledger"
)

func TestMerkleRootDeterministic(t *testing.T) {

	txs := []ledger.Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 10},
		{Sender: "Bob", Receiver: "Charlie", Amount: 5},
	}

	root1 := MerkleRoot(txs)
	root2 := MerkleRoot(txs)

	if root1 != root2 {
		t.Error("Merkle root should be deterministic for the same input")
	}
}

func TestMerkleRootChangesWithTransactionData(t *testing.T) {

	txs := []ledger.Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 10},
	}

	before := MerkleRoot(txs)

	txs[0].Amount = 999

	after := MerkleRoot(txs)

	if before == after {
		t.Error("Merkle root should change when a transaction changes")
	}
}

func TestMerkleRootHandlesOddCount(t *testing.T) {

	txs := []ledger.Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 10},
		{Sender: "Bob", Receiver: "Charlie", Amount: 5},
		{Sender: "Charlie", Receiver: "Dave", Amount: 3},
	}

	root := MerkleRoot(txs)

	if root == "" {
		t.Error("Merkle root should not be empty for a non-empty transaction list")
	}
}

func TestMerkleRootEmptyIsDeterministic(t *testing.T) {

	root1 := MerkleRoot([]ledger.Transaction{})
	root2 := MerkleRoot(nil)

	if root1 != root2 {
		t.Error("empty and nil transaction lists should produce the same root")
	}
}
