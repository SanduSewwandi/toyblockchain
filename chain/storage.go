package chain

import (
	"encoding/json"
	"fmt"
	"os"
)

// SaveToFile writes the blockchain to a JSON file.
func (bc *Blockchain) SaveToFile(filename string) error {

	data, err := json.MarshalIndent(bc, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize blockchain: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to save blockchain: %w", err)
	}

	return nil
}

// LoadFromFile loads the blockchain from a JSON file.
// If the file does not exist, a new blockchain containing
// the genesis block is created and saved.
func LoadFromFile(filename string) (*Blockchain, error) {

	// First run: create a new blockchain.
	if _, err := os.Stat(filename); os.IsNotExist(err) {

		bc := NewBlockchain()

		// Save the genesis block immediately.
		if err := bc.SaveToFile(filename); err != nil {
			return nil, err
		}

		return bc, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read blockchain: %w", err)
	}

	var bc Blockchain

	// Parse JSON.
	if err := json.Unmarshal(data, &bc); err != nil {
		return nil, fmt.Errorf("failed to parse blockchain: %w", err)
	}

	// Safety check.
	if len(bc.Blocks) == 0 {
		return NewBlockchain(), nil
	}

	return &bc, nil
}
