package chain

import (
	"strings"
	"testing"

	"toyblockchain/ledger"
)

// Test valid blockchain
func TestValidateValidChain(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	valid, message := bc.ValidateChain()

	if !valid {

		t.Errorf(
			"Expected valid chain, got invalid: %s",
			message,
		)
	}
}

// Test transaction tampering detection
func TestValidateDetectsTransactionTampering(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	// Modify transaction without updating MerkleRoot
	bc.Blocks[1].Transactions[0].Amount = 999

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should be invalid after transaction tampering")
	}

	if !strings.Contains(msg, "Merkle root mismatch") &&
		!strings.Contains(msg, "hash mismatch") {

		t.Errorf(
			"expected tampering detection, got: %s",
			msg,
		)
	}
}

// Test Merkle root tampering
func TestValidateDetectsMerkleRootTampering(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	bc.Blocks[1].MerkleRoot =
		"fake_merkle_root"

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail after Merkle root tampering")
	}

	if !strings.Contains(msg, "Merkle root mismatch") {

		t.Errorf(
			"expected Merkle root mismatch, got: %s",
			msg,
		)
	}
}

// Test invalid previous hash link
func TestValidateInvalidPreviousHash(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	bc.Blocks[1].PreviousHash =
		"wrong_hash"

	bc.Blocks[1].Hash =
		bc.Blocks[1].CalculateHash()

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail previous hash validation")
	}

	if !strings.Contains(msg, "invalid previous hash link") {

		t.Errorf(
			"expected previous hash failure, got: %s",
			msg,
		)
	}
}

// Test invalid block index
func TestValidateInvalidIndex(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	bc.Blocks[1].Index = 5

	bc.Blocks[1].Hash =
		bc.Blocks[1].CalculateHash()

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail index validation")
	}

	if !strings.Contains(msg, "invalid block index") {

		t.Errorf(
			"expected index failure, got: %s",
			msg,
		)
	}
}

// Test invalid timestamp
func TestValidateInvalidTimestamp(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	bc.Blocks[1].Timestamp =
		bc.Blocks[0].Timestamp - 100

	bc.Blocks[1].Hash =
		bc.Blocks[1].CalculateHash()

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail timestamp validation")
	}

	if !strings.Contains(msg, "invalid timestamp") {

		t.Errorf(
			"expected timestamp failure, got: %s",
			msg,
		)
	}
}

// Test that a block claiming a difficulty other than what retargeting
// expects at its position is rejected, independent of whether its PoW
// or minimum-difficulty checks would otherwise pass.
func TestValidateRejectsUnexpectedDifficulty(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	bc.Blocks[1].Difficulty = MinDifficulty
	MineBlock(&bc.Blocks[1], MinDifficulty)

	valid, msg := bc.ValidateChain()

	if valid {
		t.Fatal("expected chain with unexpected difficulty to be rejected")
	}

	if !strings.Contains(msg, "invalid difficulty") {
		t.Errorf(
			"expected invalid difficulty message, got: %s",
			msg,
		)
	}
}

func TestValidateInvalidProofOfWork(t *testing.T) {

	bc := NewBlockchain()

	tx := createSignedTransaction(
		"Alice",
		"Bob",
		10,
	)

	if err := bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	); err != nil {
		t.Fatal(err)
	}

	// Tamper the hash so it no longer satisfies its own (correct,
	// expected) difficulty's proof-of-work target, without touching
	// Index, PreviousHash, Timestamp, or Difficulty.
	bc.Blocks[1].Hash = strings.Repeat("f", len(bc.Blocks[1].Hash))

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail proof-of-work validation")
	}

	if !strings.Contains(msg, "hash mismatch") &&
		!strings.Contains(msg, "invalid proof-of-work") {

		t.Errorf(
			"expected proof-of-work or hash-mismatch failure, got: %s",
			msg,
		)
	}
}

// Test overspending detection. AddBlock now validates transactions
// against replayed ledger state before mining, so an overspend must be
// rejected at AddBlock time — it should never make it into the chain
// for ValidateChain to catch after the fact.
func TestValidateDetectsOverspendInChain(t *testing.T) {

	bc := NewBlockchain()

	badTx := createSignedTransaction(
		"Alice",
		"Mallory",
		999999,
	)

	err := bc.AddBlock(
		[]ledger.Transaction{badTx},
		DefaultDifficulty,
	)

	if err == nil {
		t.Fatal("expected AddBlock to reject an overspending transaction")
	}

	if len(bc.Blocks) != 1 {
		t.Fatalf(
			"expected chain to remain at genesis only after a rejected block, got %d blocks",
			len(bc.Blocks),
		)
	}

	// The chain should still be perfectly valid, since nothing bad
	// was ever appended.
	valid, msg := bc.ValidateChain()

	if !valid {
		t.Errorf(
			"expected chain to remain valid after rejected overspend, got invalid: %s",
			msg,
		)
	}
}
