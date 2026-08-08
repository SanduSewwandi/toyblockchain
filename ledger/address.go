package ledger

import "toyblockchain/crypto"

func (tx Transaction) SenderAddress() string {

	if tx.Sender == "" {
		return ""
	}

	return crypto.AddressFromPublicKeyHex(tx.PublicKey)
}
