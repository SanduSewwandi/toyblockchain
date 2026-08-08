package ledger

import (
	"testing"

	"toyblockchain/crypto"
)

func TestSenderAddressDerivedFromPublicKey(t *testing.T) {

	kp, _ := crypto.GenerateKeyPair()

	tx := Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   10,
	}

	SignTransaction(&tx, kp)

	if tx.SenderAddress() != kp.PublicKeyHex() {
		t.Errorf(
			"expected sender address %s, got %s",
			kp.PublicKeyHex(),
			tx.SenderAddress(),
		)
	}
}

func TestSenderAddressEmptyForCoinbase(t *testing.T) {

	tx := Transaction{Sender: "", Receiver: "Alice", Amount: 100}

	if tx.SenderAddress() != "" {
		t.Error("expected empty sender address for coinbase transaction")
	}
}
