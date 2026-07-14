package crypto

import (
	"crypto/ed25519"
	"encoding/hex"
)

// Sign signs a message with this key pair's private key, returning a
// hex-encoded signature suitable for storing on a Transaction.
func (kp KeyPair) Sign(message []byte) string {

	sig := ed25519.Sign(kp.PrivateKey, message)
	return hex.EncodeToString(sig)
}

// VerifySignature checks whether sigHex is a valid Ed25519 signature over
// message, produced by the private key corresponding to publicKeyHex.
// Returns false (never an error) on any malformed input, since a malformed
// signature should simply fail verification, not panic or bubble an error.
func VerifySignature(publicKeyHex string, message []byte, sigHex string) bool {

	pubBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(pubBytes), message, sigBytes)
}
