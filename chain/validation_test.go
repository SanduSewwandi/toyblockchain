package chain

import (
	"strings"
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

// Test valid blockchain
func TestValidateValidChain(t *testing.T) {

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

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	)

	// Modify transaction without updating MerkleRoot or Hash
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

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	)

	// Change stored Merkle root
	bc.Blocks[1].MerkleRoot = "fake_merkle_root"

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

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	)

	bc.Blocks[1].PreviousHash = "wrong_hash"

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

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	)

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

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	bc.AddBlock(
		[]ledger.Transaction{tx},
		DefaultDifficulty,
	)

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

// Test invalid proof of work
func TestValidateInvalidProofOfWork(t *testing.T) {

	bc := NewBlockchain()

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	difficulty := 3

	b := block.NewBlock(
		1,
		[]ledger.Transaction{tx},
		bc.Blocks[0].Hash,
		difficulty,
	)

	// Do not mine
	b.Hash = b.CalculateHash()

	if strings.HasPrefix(
		b.Hash,
		strings.Repeat("0", difficulty),
	) {

		t.Skip(
			"Nonce 0 unexpectedly satisfied difficulty",
		)
	}

	bc.Blocks = append(
		bc.Blocks,
		b,
	)

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail proof-of-work validation")
	}

	if !strings.Contains(msg, "invalid proof-of-work") {

		t.Errorf(
			"expected proof-of-work failure, got: %s",
			msg,
		)
	}
}

// Test overspending detection
func TestValidateDetectsOverspendInChain(t *testing.T) {

	bc := NewBlockchain()

	badTx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Mallory",
		Amount:   999999,
	}

	b := block.NewBlock(
		1,
		[]ledger.Transaction{badTx},
		bc.Blocks[0].Hash,
		DefaultDifficulty,
	)

	MineBlock(
		&b,
		DefaultDifficulty,
	)

	bc.Blocks = append(
		bc.Blocks,
		b,
	)

	valid, msg := bc.ValidateChain()

	if valid {

		t.Error(
			"chain with overspending transaction should fail",
		)
	}

	t.Log(
		"validation message:",
		msg,
	)
}
