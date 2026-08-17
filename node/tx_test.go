package node

import (
	"testing"

	"toyblockchain/ledger"
)

// FR-2: signed transactions are verified before acceptance.

func TestAddTransactionAcceptsValidSignedTransaction(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	accepted, err := n.AddTransaction(tx)

	if err != nil {
		t.Fatalf("expected valid transaction to be accepted, got error: %v", err)
	}

	if !accepted {
		t.Fatal("expected valid transaction to be accepted")
	}

	if n.PendingCount() != 1 {
		t.Fatalf("expected 1 pending transaction, got %d", n.PendingCount())
	}
}

func TestAddTransactionRejectsInvalidSignature(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	// Tamper the amount after signing so the signature no longer
	// covers the actual transaction contents.
	tx.Amount = 999

	accepted, err := n.AddTransaction(tx)

	if accepted {
		t.Fatal("expected transaction with invalid signature to be rejected")
	}

	if err == nil {
		t.Fatal("expected an error for invalid signature")
	}

	if n.PendingCount() != 0 {
		t.Fatalf("expected 0 pending transactions after rejection, got %d", n.PendingCount())
	}
}

func TestAddTransactionRejectsWrongPublicKeyForSender(t *testing.T) {

	n := NewNode(":0", nil)

	// Register Alice's key with a first, valid transaction.
	tx1 := newSignedTx(t, "Alice", "Bob", 10)

	if accepted, err := n.AddTransaction(tx1); !accepted || err != nil {
		t.Fatalf("setup transaction should be accepted, got accepted=%v err=%v", accepted, err)
	}

	// A second "Alice" transaction signed by a different key pair
	// must be rejected: the sender name is already bound to the
	// first key.
	tx2 := newSignedTx(t, "Alice", "Charlie", 5)

	accepted, err := n.AddTransaction(tx2)

	if accepted {
		t.Fatal("expected transaction signed by an unregistered key for an existing sender to be rejected")
	}

	if err == nil {
		t.Fatal("expected an error for public key mismatch")
	}
}

// FR-2 acceptance criterion: sender address is derived from the
// signing public key, not trusted as free-form input.

func TestSenderAddressDerivedFromPublicKeyInAcceptedTransaction(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	if accepted, err := n.AddTransaction(tx); !accepted || err != nil {
		t.Fatalf("expected transaction to be accepted, got accepted=%v err=%v", accepted, err)
	}

	if tx.SenderAddress() != tx.PublicKey {
		t.Errorf(
			"expected sender address to equal the signing public key, got %s vs %s",
			tx.SenderAddress(),
			tx.PublicKey,
		)
	}
}

func TestAddTransactionRejectsEmptySender(t *testing.T) {

	n := NewNode(":0", nil)

	tx := ledger.Transaction{
		Sender:   "",
		Receiver: "Mallory",
		Amount:   999999,
	}

	accepted, err := n.AddTransaction(tx)

	if accepted {
		t.Fatal("expected empty-sender transaction to be rejected")
	}

	if err == nil {
		t.Fatal("expected an error for empty sender")
	}

	if n.PendingCount() != 0 {
		t.Fatalf("expected 0 pending transactions, got %d", n.PendingCount())
	}

	if bal := n.Balance("Mallory"); bal != 0 {
		t.Errorf("expected Mallory's balance to remain 0, got %d", bal)
	}
}

func TestAddTransactionRejectsUnsignedTransaction(t *testing.T) {

	n := NewNode(":0", nil)

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
		// Signature and PublicKey left empty.
	}

	accepted, err := n.AddTransaction(tx)

	if accepted {
		t.Fatal("expected unsigned transaction to be rejected")
	}

	if err == nil {
		t.Fatal("expected an error for unsigned transaction")
	}
}

// FR-3: de-duplication.

func TestAddTransactionDeduplicates(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	accepted1, err1 := n.AddTransaction(tx)

	if !accepted1 || err1 != nil {
		t.Fatalf("expected first submission to be accepted, got accepted=%v err=%v", accepted1, err1)
	}

	accepted2, err2 := n.AddTransaction(tx)

	if accepted2 {
		t.Fatal("expected duplicate transaction to be rejected as a duplicate")
	}

	if err2 != nil {
		t.Fatalf("expected duplicate rejection to be silent (nil error), got: %v", err2)
	}

	if n.PendingCount() != 1 {
		t.Fatalf("expected pending pool to still contain exactly 1 transaction, got %d", n.PendingCount())
	}
}

// FR-9: mempool consistency — mined transactions are removed from
// the pending pool, and double-spend from the pool itself is
// prevented.

func TestRemoveMinedTransactionsClearsPendingPool(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	if accepted, err := n.AddTransaction(tx); !accepted || err != nil {
		t.Fatalf("setup: expected transaction to be accepted, got accepted=%v err=%v", accepted, err)
	}

	if n.PendingCount() != 1 {
		t.Fatalf("setup: expected 1 pending transaction, got %d", n.PendingCount())
	}

	n.RemoveMinedTransactions([]ledger.Transaction{tx})

	if n.PendingCount() != 0 {
		t.Fatalf("expected mined transaction to be removed from the pending pool, got %d remaining", n.PendingCount())
	}
}

func TestAddTransactionPreventsDoubleSpendFromPendingPool(t *testing.T) {

	n := NewNode(":0", nil)

	// Alice starts with 100 (genesis). Spend 90 of it in one pending tx.
	tx1 := newSignedTx(t, "Alice", "Bob", 90)

	if accepted, err := n.AddTransaction(tx1); !accepted || err != nil {
		t.Fatalf("setup: expected first transaction to be accepted, got accepted=%v err=%v", accepted, err)
	}

	// A second transaction spending another 90 should be rejected:
	// pendingLedgerLocked replays tx1 first, leaving Alice with only
	// 10 available.
	tx2 := newSignedTx(t, "Alice", "Charlie", 90)

	accepted, err := n.AddTransaction(tx2)

	if accepted {
		t.Fatal("expected second transaction to be rejected as an overspend against the pending pool")
	}

	if err == nil {
		t.Fatal("expected an error for overspending against pending transactions")
	}

	if n.PendingCount() != 1 {
		t.Fatalf("expected pending pool to still contain exactly 1 transaction, got %d", n.PendingCount())
	}
}
