package node

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"toyblockchain/crypto"
)

type storedIdentity struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func LoadOrCreateIdentity(path string) (crypto.KeyPair, error) {

	if _, err := os.Stat(path); os.IsNotExist(err) {

		kp, err := crypto.GenerateKeyPair()
		if err != nil {
			return crypto.KeyPair{}, fmt.Errorf("failed to generate node identity: %w", err)
		}

		if err := saveIdentity(path, kp); err != nil {
			return crypto.KeyPair{}, err
		}

		return kp, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return crypto.KeyPair{}, fmt.Errorf("failed to read identity file: %w", err)
	}

	var stored storedIdentity

	if err := json.Unmarshal(data, &stored); err != nil {
		return crypto.KeyPair{}, fmt.Errorf("failed to parse identity file: %w", err)
	}

	return crypto.KeyPairFromHex(stored.PublicKey, stored.PrivateKey)
}

func saveIdentity(path string, kp crypto.KeyPair) error {

	stored := storedIdentity{
		PublicKey:  kp.PublicKeyHex(),
		PrivateKey: kp.PrivateKeyHex(),
	}

	data, err := json.MarshalIndent(stored, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize identity: %w", err)
	}

	dir := filepath.Dir(path)

	if dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("failed to create identity directory: %w", err)
		}
	}

	return writeIdentityFileAtomic(path, data)
}

func writeIdentityFileAtomic(filename string, data []byte) error {

	dir := filepath.Dir(filename)

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Chmod(tmpName, 0600); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file into place: %w", err)
	}

	return nil
}
