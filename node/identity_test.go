package node

import (
	"path/filepath"
	"testing"
)

func TestLoadOrCreateIdentityGeneratesAndPersists(t *testing.T) {

	dir := t.TempDir()
	path := filepath.Join(dir, "identity.json")

	kp1, err := LoadOrCreateIdentity(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if kp1.PublicKeyHex() == "" {
		t.Fatal("expected a non-empty public key for a freshly generated identity")
	}

	// A second call for the same path must return the SAME identity,
	// not generate a new one — this is the "reload on restart" case.
	kp2, err := LoadOrCreateIdentity(path)
	if err != nil {
		t.Fatalf("unexpected error on reload: %v", err)
	}

	if kp1.PublicKeyHex() != kp2.PublicKeyHex() {
		t.Fatal("expected the same identity to be returned across restarts")
	}
}

func TestLoadOrCreateIdentityCreatesParentDirectory(t *testing.T) {

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "identity.json")

	if _, err := LoadOrCreateIdentity(path); err != nil {
		t.Fatalf("expected identity file to be created in a nested directory, got error: %v", err)
	}
}

func TestLoadOrCreateIdentityRejectsCorruptFile(t *testing.T) {

	dir := t.TempDir()
	path := filepath.Join(dir, "identity.json")

	if err := writeIdentityFileAtomic(path, []byte("not valid json")); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}

	if _, err := LoadOrCreateIdentity(path); err == nil {
		t.Fatal("expected an error loading a corrupt identity file")
	}
}
