package wallet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetOrCreateGeneratesAndPersists(t *testing.T) {

	dir := t.TempDir()
	file := filepath.Join(dir, "wallet.json")

	w := NewWallet(file)

	kp1, err := w.GetOrCreate("Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if kp1.PublicKeyHex() == "" {
		t.Error("expected a non-empty public key for a new account")
	}

	// Calling again for the same name must return the same key pair,
	// not silently generate a new one.
	kp2, err := w.GetOrCreate("Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if kp1.PublicKeyHex() != kp2.PublicKeyHex() {
		t.Error("expected GetOrCreate to return the same key pair for an existing name")
	}
}

func TestWalletSaveAndLoadRoundTrip(t *testing.T) {

	dir := t.TempDir()
	file := filepath.Join(dir, "wallet.json")

	w := NewWallet(file)

	kp, err := w.GetOrCreate("Bob")
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadWallet(file)
	if err != nil {
		t.Fatalf("unexpected error loading wallet: %v", err)
	}

	reloadedKp, err := loaded.GetOrCreate("Bob")
	if err != nil {
		t.Fatal(err)
	}

	if reloadedKp.PublicKeyHex() != kp.PublicKeyHex() {
		t.Error("expected reloaded wallet to return the same key pair for an existing account")
	}
}

func TestLoadWalletCreatesFileIfMissing(t *testing.T) {

	dir := t.TempDir()
	file := filepath.Join(dir, "does_not_exist_yet.json")

	w, err := LoadWallet(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(w.Accounts) != 0 {
		t.Error("expected a freshly created wallet to have no accounts")
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		t.Error("expected LoadWallet to create the file on disk when missing")
	}
}

func TestNameForPublicKeyReverseLookup(t *testing.T) {

	dir := t.TempDir()
	file := filepath.Join(dir, "wallet.json")

	w := NewWallet(file)

	kp, err := w.GetOrCreate("Charlie")
	if err != nil {
		t.Fatal(err)
	}

	name, ok := w.NameForPublicKey(kp.PublicKeyHex())
	if !ok {
		t.Fatal("expected to find a name for a known public key")
	}

	if name != "Charlie" {
		t.Errorf("expected name %q, got %q", "Charlie", name)
	}
}

func TestNameForPublicKeyReturnsFalseForUnknownKey(t *testing.T) {

	dir := t.TempDir()
	file := filepath.Join(dir, "wallet.json")

	w := NewWallet(file)

	_, ok := w.NameForPublicKey("deadbeef")
	if ok {
		t.Error("expected NameForPublicKey to return false for an unregistered public key")
	}
}
