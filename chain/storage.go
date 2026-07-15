package chain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func writeFileAtomic(filename string, data []byte, perm os.FileMode) error {

	dir := filepath.Dir(filename)

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpName := tmp.Name()

	// Ensure the temp file is cleaned up if anything below fails before
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file into place: %w", err)
	}

	return nil
}

// SaveToFile writes the blockchain to a JSON file.
func (bc *Blockchain) SaveToFile(filename string) error {

	data, err := json.MarshalIndent(bc, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize blockchain: %w", err)
	}

	if err := writeFileAtomic(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to save blockchain: %w", err)
	}

	return nil
}

func LoadCandidateFromFile(filename string) (*Blockchain, error) {

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read candidate chain: %w", err)
	}

	var bc Blockchain

	if err := json.Unmarshal(data, &bc); err != nil {
		return nil, fmt.Errorf("failed to parse candidate chain: %w", err)
	}

	return &bc, nil
}

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

	if valid, msg := bc.ValidateChain(); !valid {
		return nil, fmt.Errorf("blockchain.json failed validation: %s", msg)
	}

	return &bc, nil
}
