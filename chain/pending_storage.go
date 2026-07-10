package chain

import (
	"encoding/json"
	"fmt"
	"os"

	"toyblockchain/ledger"
)

// SavePending saves pending transactions to a JSON file.
func SavePending(filename string, transactions []ledger.Transaction) error {

	data, err := json.MarshalIndent(transactions, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize pending transactions: %w", err)
	}

	if err := writeFileAtomic(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to save pending transactions: %w", err)
	}

	return nil
}

// LoadPending loads pending transactions from a JSON file.
// If the file does not exist, an empty pending transaction pool is created.
func LoadPending(filename string) ([]ledger.Transaction, error) {

	if _, err := os.Stat(filename); os.IsNotExist(err) {

		empty := []ledger.Transaction{}

		if err := SavePending(filename, empty); err != nil {
			return nil, err
		}

		return empty, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read pending transactions: %w", err)
	}

	var transactions []ledger.Transaction

	if err := json.Unmarshal(data, &transactions); err != nil {
		return nil, fmt.Errorf("failed to parse pending transactions: %w", err)
	}

	return transactions, nil
}

// ClearPending removes all pending transactions.
func ClearPending(filename string) error {

	return SavePending(
		filename,
		[]ledger.Transaction{},
	)
}
