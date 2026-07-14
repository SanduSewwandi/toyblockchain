package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// KeyPair holds an Ed25519 public/private key pair for one account.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair creates a new random Ed25519 key pair.
func GenerateKeyPair() (KeyPair, error) {

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return KeyPair{PublicKey: pub, PrivateKey: priv}, nil
}

// PublicKeyHex returns the hex-encoded public key. This is used as the
// account's address throughout the ledger, in place of a plain name string.
func (kp KeyPair) PublicKeyHex() string {
	return hex.EncodeToString(kp.PublicKey)
}

// PrivateKeyHex returns the hex-encoded private key, for local persistence
// only. This must never be shared or committed to version control.
func (kp KeyPair) PrivateKeyHex() string {
	return hex.EncodeToString(kp.PrivateKey)
}

// KeyPairFromHex reconstructs a KeyPair from previously stored hex strings,
// validating that both decode to the expected Ed25519 key sizes.
func KeyPairFromHex(publicHex, privateHex string) (KeyPair, error) {

	pubBytes, err := hex.DecodeString(publicHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return KeyPair{}, fmt.Errorf("invalid public key hex")
	}

	privBytes, err := hex.DecodeString(privateHex)
	if err != nil || len(privBytes) != ed25519.PrivateKeySize {
		return KeyPair{}, fmt.Errorf("invalid private key hex")
	}

	return KeyPair{
		PublicKey:  ed25519.PublicKey(pubBytes),
		PrivateKey: ed25519.PrivateKey(privBytes),
	}, nil
}
