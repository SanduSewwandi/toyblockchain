package ledger

import "fmt"

// Ledger keeps track of balances and which public key is registered to
// each sender name, so a name can't be impersonated by signing with a
// different key pair once it has transacted at least once.
type Ledger struct {
	Balances       map[string]int64
	registeredKeys map[string]string // sender name -> public key hex, first-seen wins
}

// NewLedger creates a ledger.
func NewLedger() *Ledger {

	return &Ledger{
		Balances:       make(map[string]int64),
		registeredKeys: make(map[string]string),
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
		return fmt.Errorf("insufficient balance for %s", user)
	}

	l.Balances[user] -= amount
	return nil
}

// ApplyTransaction validates and applies a transaction. For non-coinbase
// transactions, it verifies the signature is self-consistent, and that the
// signing key matches the key already registered for this sender (or
// registers it, if this is the first time the sender has transacted).
func (l *Ledger) ApplyTransaction(tx Transaction) error {

	if tx.Amount <= 0 {
		return fmt.Errorf("invalid transaction amount: must be > 0")
	}

	if tx.Sender != "" {

		if !VerifyTransactionSignature(tx) {
			return fmt.Errorf("invalid signature for sender %s", tx.Sender)
		}

		if registered, ok := l.registeredKeys[tx.Sender]; ok {

			if registered != tx.PublicKey {
				return fmt.Errorf(
					"public key mismatch for sender %s: transaction not signed by the registered account",
					tx.Sender,
				)
			}

		} else {
			l.registeredKeys[tx.Sender] = tx.PublicKey
		}

		if err := l.Debit(tx.Sender, tx.Amount); err != nil {
			return err
		}
	}

	l.Credit(tx.Receiver, tx.Amount)
	return nil
}

// ApplyBlockTransactions applies a list of transactions.
func (l *Ledger) ApplyBlockTransactions(txs []Transaction) error {

	for _, tx := range txs {
		if err := l.ApplyTransaction(tx); err != nil {
			return err
		}
	}
	return nil
}

// Clone creates an independent copy of the ledger, including registered
// keys, so pending-pool validation against a temp ledger enforces the
// same identity binding as the real chain.
func (l *Ledger) Clone() *Ledger {

	newLedger := NewLedger()

	for user, balance := range l.Balances {
		newLedger.Balances[user] = balance
	}

	for name, key := range l.registeredKeys {
		newLedger.registeredKeys[name] = key
	}

	return newLedger
}

// Print shows balances.
func (l *Ledger) Print() {

	fmt.Println("\n========== LEDGER BALANCES ==========")

	for user, balance := range l.Balances {
		fmt.Printf("%s : %d\n", user, balance)
	}
}
