package ledger

import "testing"

// TestRejectOverspending verifies that a transaction is rejected
// when the sender does not have enough balance.
func TestRejectOverspending(t *testing.T) {

	l := NewLedger()

	// Give Alice an initial balance.
	l.Credit("Alice", 100)

	tx := Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   150,
	}

	// Transaction should fail.
	err := l.ApplyTransaction(tx)

	if err == nil {
		t.Fatal("expected overspending transaction to be rejected")
	}

	// Alice's balance should remain unchanged.
	aliceBalance := l.GetBalance("Alice")

	if aliceBalance != 100 {
		t.Errorf(
			"expected Alice balance to remain 100, got %d",
			aliceBalance,
		)
	}

	// Bob should not receive any funds.
	bobBalance := l.GetBalance("Bob")

	if bobBalance != 0 {
		t.Errorf(
			"expected Bob balance to remain 0, got %d",
			bobBalance,
		)
	}
}
