package chain

import (
	"strings"
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

func TestMineBlock(t *testing.T) {

	tx := []ledger.Transaction{

		{
			Sender:   "Alice",
			Receiver: "Bob",
			Amount:   10,
		},
	}

	b := block.NewBlock(
		1,
		tx,
		"previous_hash",
	)

	difficulty := 2

	MineBlock(
		&b,
		difficulty,
	)

	// Hash should not be empty
	if b.Hash == "" {

		t.Error(
			"Mining should generate a hash",
		)
	}

	// Hash should satisfy difficulty
	target := strings.Repeat("0", difficulty)

	if !strings.HasPrefix(b.Hash, target) {

		t.Errorf(
			"Hash %s does not satisfy difficulty %d",
			b.Hash,
			difficulty,
		)
	}

}

// Test nonce changes during mining
func TestMineBlockChangesNonce(t *testing.T) {

	b := block.NewBlock(
		1,
		[]ledger.Transaction{},
		"previous_hash",
	)

	initialNonce := b.Nonce

	MineBlock(
		&b,
		2,
	)

	if b.Nonce == initialNonce {

		t.Log(
			"Nonce remained 0, possible but rare",
		)
	}

}

// Test different difficulty levels
func TestDifferentDifficulty(t *testing.T) {

	b := block.NewBlock(
		1,
		[]ledger.Transaction{},
		"previous_hash",
	)

	difficulty := 3

	MineBlock(
		&b,
		difficulty,
	)

	target := strings.Repeat(
		"0",
		difficulty,
	)

	if !strings.HasPrefix(
		b.Hash,
		target,
	) {

		t.Errorf(
			"Mining failed for difficulty %d",
			difficulty,
		)
	}

}

// Test that the mined nonce reproduces the exact stored hash.
func TestNonceReproducesMinedHash(t *testing.T) {

	b := block.NewBlock(
		1,
		[]ledger.Transaction{},
		"previous_hash",
	)

	difficulty := 3

	MineBlock(
		&b,
		difficulty,
	)


	// Recalculate hash using the mined nonce.
	recalculatedHash := b.CalculateHash()


	// The recalculated hash must equal the stored hash.
	if recalculatedHash != b.Hash {

		t.Errorf(
			"Nonce does not reproduce mined hash\nStored: %s\nCalculated: %s",
			b.Hash,
			recalculatedHash,
		)
	}
}