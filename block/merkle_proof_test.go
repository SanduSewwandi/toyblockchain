package block

import (
	"testing"

	"toyblockchain/ledger"
)

func testTxs() []ledger.Transaction {
	return []ledger.Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 10},
		{Sender: "Bob", Receiver: "Charlie", Amount: 5},
		{Sender: "Charlie", Receiver: "Dave", Amount: 3},
	}
}

func TestGenerateMerkleProofVerifiesAgainstRoot(t *testing.T) {

	txs := testTxs()
	root := MerkleRoot(txs)

	for i := range txs {

		proof, err := GenerateMerkleProof(txs, i)
		if err != nil {
			t.Fatalf("unexpected error for index %d: %v", i, err)
		}

		if proof.Root != root {
			t.Fatalf("proof root %s does not match block MerkleRoot %s", proof.Root, root)
		}

		if !VerifyMerkleProof(proof) {
			t.Errorf("expected proof for tx %d to verify", i)
		}
	}
}

func TestVerifyMerkleProofRejectsTamperedTxHash(t *testing.T) {

	txs := testTxs()

	proof, err := GenerateMerkleProof(txs, 1)
	if err != nil {
		t.Fatal(err)
	}

	proof.TxHash = "0000000000000000000000000000000000000000000000000000000000000000"

	if VerifyMerkleProof(proof) {
		t.Error("expected proof with tampered transaction hash to fail verification")
	}
}

func TestVerifyMerkleProofRejectsTamperedPath(t *testing.T) {

	txs := testTxs()

	proof, err := GenerateMerkleProof(txs, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(proof.Path) == 0 {
		t.Fatal("expected a non-empty proof path for a multi-transaction block")
	}

	proof.Path[0].Hash = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"

	if VerifyMerkleProof(proof) {
		t.Error("expected proof with tampered sibling hash to fail verification")
	}
}

func TestGenerateMerkleProofRejectsOutOfRangeIndex(t *testing.T) {

	txs := testTxs()

	if _, err := GenerateMerkleProof(txs, len(txs)); err == nil {
		t.Error("expected an error for an out-of-range transaction index")
	}

	if _, err := GenerateMerkleProof(txs, -1); err == nil {
		t.Error("expected an error for a negative transaction index")
	}
}

func TestGenerateMerkleProofSingleTransaction(t *testing.T) {

	txs := []ledger.Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 10},
	}

	proof, err := GenerateMerkleProof(txs, 0)
	if err != nil {
		t.Fatal(err)
	}

	if !VerifyMerkleProof(proof) {
		t.Error("expected single-transaction proof to verify")
	}

	if proof.Root != MerkleRoot(txs) {
		t.Error("expected proof root to match block MerkleRoot for single-transaction block")
	}
}
