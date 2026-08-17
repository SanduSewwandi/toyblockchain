package chain

import (
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
	"toyblockchain/wallet"
)

func createChainWithBlocks(
	t *testing.T,
	base *Blockchain,
	blocks int,
) *Blockchain {

	t.Helper()

	// Copy genesis block so both chains have the same genesis.
	bc := &Blockchain{
		Blocks: append(
			[]block.Block{},
			base.Blocks...,
		),
	}

	// Create temporary wallet
	w := wallet.NewWallet("test_wallet.json")

	aliceKeys, err := w.GetOrCreate("Alice")

	if err != nil {
		t.Fatalf(
			"failed creating wallet: %v",
			err,
		)
	}

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   1,
	}

	ledger.SignTransaction(
		&tx,
		aliceKeys,
	)

	for i := 1; i < blocks; i++ {

		if err := bc.AddBlock(
			[]ledger.Transaction{tx},
			DefaultDifficulty,
		); err != nil {

			t.Fatalf(
				"failed adding block: %v",
				err,
			)
		}
	}

	return bc
}

func TestResolveForkAcceptsLongerChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(
		t,
		base,
		3,
	)

	candidate := createChainWithBlocks(
		t,
		base,
		5,
	)

	accepted, msg := current.ResolveFork(candidate)

	t.Log(msg)

	if !accepted {
		t.Fatal(
			"expected longer chain to be accepted",
		)
	}

	if len(current.Blocks) != 5 {

		t.Fatalf(
			"expected 5 blocks, got %d",
			len(current.Blocks),
		)
	}
}

func TestResolveForkRejectsShorterChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(
		t,
		base,
		5,
	)

	candidate := createChainWithBlocks(
		t,
		base,
		3,
	)

	accepted, msg := current.ResolveFork(candidate)

	t.Log(msg)

	if accepted {
		t.Fatal(
			"expected shorter chain rejected",
		)
	}

	if len(current.Blocks) != 5 {

		t.Fatal(
			"current chain changed",
		)
	}
}

func TestResolveForkRejectsSameLengthChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(
		t,
		base,
		4,
	)

	candidate := createChainWithBlocks(
		t,
		base,
		4,
	)

	accepted, msg := current.ResolveFork(candidate)

	t.Log(msg)

	if accepted {

		t.Fatal(
			"expected same length chain rejected",
		)
	}
}

func TestResolveForkRejectsInvalidChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(
		t,
		base,
		3,
	)

	candidate := createChainWithBlocks(
		t,
		base,
		4,
	)

	// Tamper candidate
	candidate.Blocks[2].PreviousHash = "invalid"

	accepted, msg := current.ResolveFork(candidate)

	t.Log(msg)

	if accepted {

		t.Fatal(
			"expected invalid chain rejected",
		)
	}

	if len(current.Blocks) != 3 {

		t.Fatal(
			"current chain changed",
		)
	}
}
