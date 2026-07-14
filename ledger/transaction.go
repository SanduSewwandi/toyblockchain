package ledger

import "fmt"

// Transaction represents a transfer of value between two accounts.
type Transaction struct {
	Sender    string
	Receiver  string
	Amount    int64
	PublicKey string
	Signature string
}

func (tx Transaction) SigningBytes() []byte {

	return []byte(fmt.Sprintf(
		"%s:%s:%d",
		tx.Sender,
		tx.Receiver,
		tx.Amount,
	))
}
