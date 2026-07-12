package ledger

// Transaction represents a transfer of value between two accounts.
type Transaction struct {
	Sender   string
	Receiver string
	Amount   int64
}
