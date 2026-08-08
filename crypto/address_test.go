package crypto

import "testing"

func TestAddressMatchesPublicKeyHex(t *testing.T) {

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if kp.Address() != kp.PublicKeyHex() {
		t.Errorf(
			"expected address to equal public key hex, got %s vs %s",
			kp.Address(),
			kp.PublicKeyHex(),
		)
	}
}

func TestAddressFromPublicKeyHexMatchesKeyPair(t *testing.T) {

	kp, _ := GenerateKeyPair()

	if AddressFromPublicKeyHex(kp.PublicKeyHex()) != kp.Address() {
		t.Error("AddressFromPublicKeyHex should match KeyPair.Address()")
	}
}
