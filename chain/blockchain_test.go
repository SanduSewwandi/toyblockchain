package chain

import (
	"testing"
)

// TestGenesisCreatesInitialBalances verifies that the genesis block
// correctly creates the initial balances through SYSTEM transactions.
func TestGenesisCreatesInitialBalances(t *testing.T) {

	bc := NewBlockchain()

	ld := bc.BuildLedger()

	// Alice should receive 100 from the genesis block.
	if ld.GetBalance("Alice") != 100 {

		t.Errorf(
			"expected Alice balance 100, got %d",
			ld.GetBalance("Alice"),
		)
	}

	// Bob should receive 50 from the genesis block.
	if ld.GetBalance("Bob") != 50 {

		t.Errorf(
			"expected Bob balance 50, got %d",
			ld.GetBalance("Bob"),
		)
	}

	// Charlie should have no balance.
	if ld.GetBalance("Charlie") != 0 {

		t.Errorf(
			"expected Charlie balance 0, got %d",
			ld.GetBalance("Charlie"),
		)
	}
}

// TestGenesisIsDeterministic verifies that the genesis block is fully
// deterministic: two independently created blockchains must produce
// byte-identical genesis blocks, including the same timestamp and hash.
func TestGenesisIsDeterministic(t *testing.T) {

	bc1 := NewBlockchain()
	bc2 := NewBlockchain()

	// Genesis timestamps must match exactly.
	if bc1.Blocks[0].Timestamp != bc2.Blocks[0].Timestamp {

		t.Errorf(
			"expected identical genesis timestamps, got %d and %d",
			bc1.Blocks[0].Timestamp,
			bc2.Blocks[0].Timestamp,
		)
	}

	// Genesis timestamp must equal the fixed constant, not the current time.
	if bc1.Blocks[0].Timestamp != GenesisTimestamp {

		t.Errorf(
			"expected fixed genesis timestamp %d, got %d",
			GenesisTimestamp,
			bc1.Blocks[0].Timestamp,
		)
	}

	// Genesis hashes must match exactly.
	if bc1.Blocks[0].Hash != bc2.Blocks[0].Hash {

		t.Errorf(
			"expected identical genesis hashes, got %s and %s",
			bc1.Blocks[0].Hash,
			bc2.Blocks[0].Hash,
		)
	}
}
