package node

import (
	"testing"

	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/crypto"
	"toyblockchain/ledger"
)

// newSignedTx builds a Transaction signed by a freshly generated key
// pair, mirroring chain.createSignedTransaction so node tests don't
// need to depend on the chain package's internal test helper.
func newSignedTx(t *testing.T, sender, receiver string, amount int64) ledger.Transaction {
	t.Helper()

	kp, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	tx := ledger.Transaction{
		Sender:   sender,
		Receiver: receiver,
		Amount:   amount,
	}

	ledger.SignTransaction(&tx, kp)

	return tx
}

// mineNextBlock mines a block that correctly extends n's current tip,
// using the difficulty the node itself would expect next. Tests use
// this so that AddBlock's difficulty and PoW checks pass by default;
// individual tests then tamper with the result to exercise rejection
// paths.
func mineNextBlock(t *testing.T, n *Node, txs []ledger.Transaction) block.Block {
	t.Helper()

	n.mu.RLock()
	latest := n.Blockchain.GetLatestBlock()
	difficulty := chain.NextDifficultyFor(n.Blockchain, chain.DefaultDifficulty)
	n.mu.RUnlock()

	b := block.NewBlock(latest.Index+1, txs, latest.Hash, difficulty)
	chain.MineBlock(&b, difficulty)

	return b
}
