package crypto

import "testing"

func TestGenerateKeyPairProducesValidKeys(t *testing.T) {

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("unexpected error generating key pair: %v", err)
	}

	if kp.PublicKeyHex() == "" {
		t.Error("expected non-empty public key hex")
	}
}

func TestSignAndVerifyRoundTrip(t *testing.T) {

	kp, _ := GenerateKeyPair()

	message := []byte("Alice pays Bob 10")
	sig := kp.Sign(message)

	if !VerifySignature(kp.PublicKeyHex(), message, sig) {
		t.Error("expected valid signature to verify successfully")
	}
}

func TestVerifyFailsOnTamperedMessage(t *testing.T) {

	kp, _ := GenerateKeyPair()

	message := []byte("Alice pays Bob 10")
	sig := kp.Sign(message)

	tampered := []byte("Alice pays Bob 99999")

	if VerifySignature(kp.PublicKeyHex(), tampered, sig) {
		t.Error("expected signature verification to fail on tampered message")
	}
}

func TestVerifyFailsWithWrongPublicKey(t *testing.T) {

	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()

	message := []byte("Alice pays Bob 10")
	sig := kp1.Sign(message)

	if VerifySignature(kp2.PublicKeyHex(), message, sig) {
		t.Error("expected signature to fail verification against the wrong public key")
	}
}

func TestVerifyHandlesMalformedInputGracefully(t *testing.T) {

	if VerifySignature("not-hex-!!", []byte("msg"), "also-not-hex") {
		t.Error("expected malformed input to fail verification, not panic or succeed")
	}
}

func TestKeyPairFromHexRoundTrip(t *testing.T) {

	kp, _ := GenerateKeyPair()

	restored, err := KeyPairFromHex(kp.PublicKeyHex(), kp.PrivateKeyHex())
	if err != nil {
		t.Fatalf("unexpected error reconstructing key pair: %v", err)
	}

	message := []byte("test message")
	sig := restored.Sign(message)

	if !VerifySignature(kp.PublicKeyHex(), message, sig) {
		t.Error("reconstructed key pair should sign messages verifiable by the original public key")
	}
}

func TestKeyPairFromHexRejectsInvalidInput(t *testing.T) {

	_, err := KeyPairFromHex("not-hex", "also-not-hex")
	if err == nil {
		t.Error("expected an error for invalid hex input")
	}
}
