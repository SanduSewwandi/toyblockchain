package ledger

import "toyblockchain/crypto"

func SignTransaction(tx *Transaction, wallet crypto.KeyPair) {

	tx.PublicKey = wallet.PublicKeyHex()
	tx.Signature = wallet.Sign(tx.SigningBytes())
}
