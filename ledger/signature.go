package ledger

import "toyblockchain/crypto"

func SignTransaction(tx *Transaction, wallet crypto.KeyPair) {

	tx.Signature = wallet.Sign(tx.SigningBytes())
	tx.PublicKey = wallet.PublicKeyHex()
}
