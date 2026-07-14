package wallet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"toyblockchain/crypto"
)

// DefaultWalletFile is where named account key pairs are persisted.
//
// SECURITY NOTE: this stores private keys in plaintext JSON on disk. That
// is acceptable for a toy/educational project but is not how a production
// system would manage keys — a real wallet would use an OS keychain,
// hardware security module, or at minimum encrypt the file at rest. This
// file must never be committed to version control (see .gitignore).
const DefaultWalletFile = "wallet.json"

// storedKeyPair is the JSON-serializable form of a crypto.KeyPair.
type storedKeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// Wallet maps human-readable account names (e.g. "Alice") to key pairs,
// so the CLI can keep accepting friendly names while the ledger itself
// deals in real public keys as addresses.
type Wallet struct {
	Accounts map[string]storedKeyPair `json:"accounts"`
	filename string
}

// NewWallet creates an empty, unsaved wallet bound to filename.
func NewWallet(filename string) *Wallet {

	return &Wallet{
		Accounts: make(map[string]storedKeyPair),
		filename: filename,
	}
}

// LoadWallet loads a wallet from filename, creating a new empty one (and
// saving it) if the file doesn't exist yet.
func LoadWallet(filename string) (*Wallet, error) {

	if _, err := os.Stat(filename); os.IsNotExist(err) {

		w := NewWallet(filename)

		if err := w.Save(); err != nil {
			return nil, err
		}

		return w, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %w", err)
	}

	var w Wallet

	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("failed to parse wallet file: %w", err)
	}

	w.filename = filename

	if w.Accounts == nil {
		w.Accounts = make(map[string]storedKeyPair)
	}

	return &w, nil
}

// Save writes the wallet to disk atomically (temp file + rename), so a
// crash mid-write can't corrupt an existing wallet file.
func (w *Wallet) Save() error {

	data, err := json.MarshalIndent(w, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize wallet: %w", err)
	}

	return writeFileAtomic(w.filename, data, 0600)
}

// GetOrCreate returns the key pair for name, generating and persisting a
// new one the first time this name is seen. Subsequent calls for the same
// name always return the same key pair.
func (w *Wallet) GetOrCreate(name string) (crypto.KeyPair, error) {

	if stored, ok := w.Accounts[name]; ok {
		return crypto.KeyPairFromHex(stored.PublicKey, stored.PrivateKey)
	}

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		return crypto.KeyPair{}, err
	}

	w.Accounts[name] = storedKeyPair{
		PublicKey:  kp.PublicKeyHex(),
		PrivateKey: kp.PrivateKeyHex(),
	}

	if err := w.Save(); err != nil {
		return crypto.KeyPair{}, err
	}

	return kp, nil
}

// NameForPublicKey reverse-looks-up a name for a given public key hex, for
// display purposes (e.g. printing "Alice" instead of a raw hex address).
// Returns ok=false if no matching name is found.
func (w *Wallet) NameForPublicKey(publicKeyHex string) (string, bool) {

	for name, stored := range w.Accounts {

		if stored.PublicKey == publicKeyHex {
			return name, true
		}
	}

	return "", false
}

// writeFileAtomic writes data to filename without ever leaving a
// partially-written file in place, mirroring the same pattern used for
// blockchain and pending-pool persistence in the chain package.
func writeFileAtomic(filename string, data []byte, perm os.FileMode) error {

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

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	if err := os.Rename(tmpName, filename); err != nil {
		return fmt.Errorf("failed to rename temp file into place: %w", err)
	}

	return nil
}
