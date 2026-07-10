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

// Test tampering detection (hash-integrity branch — this one was always correct,
// since the transaction edit deliberately leaves Hash stale).
func TestValidateDetectsTampering(t *testing.T) {

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

	// Modify transaction data without recomputing hash — this is the
	// "attacker edits data on disk" scenario, so leaving Hash stale is correct here.
	bc.Blocks[1].Transactions[0].Amount = 999

	valid, message := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should be invalid after tampering")
	}

	if !strings.Contains(message, "hash mismatch") {
		t.Errorf("expected hash mismatch message, got: %s", message)
	}
}

// Test invalid previous hash link — recompute Hash so the tamper is caught
// by the previous-hash-link check specifically, not the earlier hash-integrity check.
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

	// Break the chain link, then recompute the hash so hash-integrity passes
	// and validation actually reaches the previous-hash-link check.
	bc.Blocks[1].PreviousHash = "wrong_hash"
	bc.Blocks[1].Hash = bc.Blocks[1].CalculateHash()

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail previous hash validation")
	}

	if !strings.Contains(msg, "invalid previous hash link") {
		t.Errorf("expected previous-hash-link failure, got: %s", msg)
	}
}

// Test invalid block index — same pattern: recompute hash after tampering.
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

	// Change block height, then recompute hash so we reach the index check.
	bc.Blocks[1].Index = 5
	bc.Blocks[1].Hash = bc.Blocks[1].CalculateHash()

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail index validation")
	}

	if !strings.Contains(msg, "invalid block index") {
		t.Errorf("expected index-validation failure, got: %s", msg)
	}
}

// Test invalid timestamp order — same pattern: recompute hash after tampering.
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

	// Set the block's timestamp earlier than its predecessor, then recompute
	// hash so we reach the timestamp check.
	bc.Blocks[1].Timestamp = bc.Blocks[0].Timestamp - 100
	bc.Blocks[1].Hash = bc.Blocks[1].CalculateHash()

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail timestamp validation")
	}

	if !strings.Contains(msg, "invalid timestamp") {
		t.Errorf("expected timestamp-validation failure, got: %s", msg)
	}
}

// Test invalid proof of work — this one can't be fixed by just recomputing
// the hash, since a hash honestly recomputed from real mined data will still
// satisfy the difficulty it was mined at. Instead, build a block directly
// (skipping MineBlock) so its nonce was never searched for, giving it a
// hash that is internally consistent (passes hash-integrity) but almost
// certainly doesn't have the required leading zeros — which is exactly what
// "claims a difficulty it never mined at" looks like.
func TestValidateInvalidProofOfWork(t *testing.T) {

	bc := NewBlockchain()

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	difficulty := 3 // must be >= MinDifficulty, or we'd hit the floor check instead

	b := block.NewBlock(
		bc.Blocks[0].Index+1,
		[]ledger.Transaction{tx},
		bc.Blocks[0].Hash,
		difficulty,
	)
	// Deliberately NOT calling MineBlock: Nonce stays at 0, so b.Hash
	// (set by NewBlock) is internally consistent but essentially random
	// with respect to the difficulty target.
	b.Hash = b.CalculateHash()

	// Guard against the astronomically unlikely case that nonce 0 happens
	// to satisfy difficulty 3 anyway (~1 in 4096).
	if strings.HasPrefix(b.Hash, strings.Repeat("0", difficulty)) {
		t.Skip("nonce 0 happened to satisfy the difficulty target by chance; rerun")
	}

	bc.Blocks = append(bc.Blocks, b)

	valid, msg := bc.ValidateChain()

	if valid {
		t.Error("Blockchain should fail proof-of-work validation")
	}

	if !strings.Contains(msg, "invalid proof-of-work") {
		t.Errorf("expected proof-of-work failure, got: %s", msg)
	}
}

func TestValidateDetectsOverspendInChain(t *testing.T) {

	bc := NewBlockchain()

	// Alice only has 100 from genesis — this overspends.
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

	MineBlock(&b, DefaultDifficulty)

	bc.Blocks = append(bc.Blocks, b)

	valid, msg := bc.ValidateChain()

	if valid {

		t.Error(
			"chain with an overspending transaction should fail validation",
		)
	}

	t.Log("validation message:", msg)
}
