package ledger

import "fmt"

// Ledger keeps track of balances.
type Ledger struct {
	Balances map[string]int64
}

// NewLedger creates a ledger.
func NewLedger() *Ledger {

	return &Ledger{
		Balances: make(map[string]int64),
	}
}

// GetBalance returns balance.
func (l *Ledger) GetBalance(user string) int64 {

	return l.Balances[user]
}

// Credit adds funds.
func (l *Ledger) Credit(user string, amount int64) {

	l.Balances[user] += amount
}

// Debit subtracts funds safely.
func (l *Ledger) Debit(user string, amount int64) error {

	if l.Balances[user] < amount {

		return fmt.Errorf(
			"insufficient balance for %s",
			user,
		)
	}

	l.Balances[user] -= amount

	return nil
}

// ApplyTransaction validates and applies a transaction.
func (l *Ledger) ApplyTransaction(tx Transaction) error {

	// Reject invalid amounts
	if tx.Amount <= 0 {

		return fmt.Errorf(
			"invalid transaction amount: must be > 0",
		)
	}

	// Sender logic (skip for minting)
	if tx.Sender != "" {

		err := l.Debit(
			tx.Sender,
			tx.Amount,
		)

		if err != nil {
			return err
		}
	}

	// Add amount to receiver
	l.Credit(
		tx.Receiver,
		tx.Amount,
	)

	return nil
}

// ApplyBlockTransactions applies a list of transactions.
func (l *Ledger) ApplyBlockTransactions(txs []Transaction) error {

	for _, tx := range txs {

		err := l.ApplyTransaction(tx)

		if err != nil {
			return err
		}
	}

	return nil
}

// Clone creates an independent copy of the ledger.
//
// This is used when validating pending transactions.
// We test transactions on a temporary ledger
// without changing the real blockchain balance.
func (l *Ledger) Clone() *Ledger {

	newLedger := NewLedger()

	for user, balance := range l.Balances {

		newLedger.Balances[user] = balance
	}

	return newLedger
}

// Print shows balances.
func (l *Ledger) Print() {

	fmt.Println("\n========== LEDGER BALANCES ==========")

	for user, balance := range l.Balances {

		fmt.Printf(
			"%s : %d\n",
			user,
			balance,
		)
	}
}