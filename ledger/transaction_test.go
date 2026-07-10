package ledger

import "testing"

// TestApplyTransactionRejectsNonPositiveAmount verifies that transactions
// with zero or negative amounts are rejected before any balance changes.
func TestApplyTransactionRejectsNonPositiveAmount(t *testing.T) {

	l := NewLedger()
	l.Credit("Alice", 100)

	cases := []Transaction{
		{Sender: "Alice", Receiver: "Bob", Amount: 0},
		{Sender: "Alice", Receiver: "Bob", Amount: -10},
	}

	for _, tx := range cases {

		err := l.ApplyTransaction(tx)

		if err == nil {
			t.Errorf(
				"expected transaction with amount %d to be rejected",
				tx.Amount,
			)
		}

		if l.GetBalance("Alice") != 100 {
			t.Errorf(
				"Alice balance should remain unchanged after rejected tx, got %d",
				l.GetBalance("Alice"),
			)
		}

		if l.GetBalance("Bob") != 0 {
			t.Errorf(
				"Bob should not receive funds from a rejected tx, got %d",
				l.GetBalance("Bob"),
			)
		}
	}
}

// TestApplyTransactionMintsWithEmptySender verifies that a transaction with
// an empty sender (the coinbase/faucet convention) credits the receiver
// without requiring or debiting any balance.
func TestApplyTransactionMintsWithEmptySender(t *testing.T) {

	l := NewLedger()

	tx := Transaction{
		Sender:   "",
		Receiver: "Alice",
		Amount:   100,
	}

	err := l.ApplyTransaction(tx)

	if err != nil {
		t.Fatalf("expected empty-sender mint to succeed, got error: %v", err)
	}

	if l.GetBalance("Alice") != 100 {
		t.Errorf("expected Alice balance 100, got %d", l.GetBalance("Alice"))
	}
}