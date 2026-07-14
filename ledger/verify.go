package ledger

import "toyblockchain/crypto"

func VerifyTransactionSignature(tx Transaction) bool {

	return crypto.VerifySignature(
		tx.PublicKey,
		tx.SigningBytes(),
		tx.Signature,
	)
}
