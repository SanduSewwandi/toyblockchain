package chain

import (
	"strings"
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

func TestMineBlockConcurrentSatisfiesDifficulty(t *testing.T) {

	difficulty := 3

	b := block.NewBlock(
		1,
		[]ledger.Transaction{
			{Sender: "Alice", Receiver: "Bob", Amount: 10},
		},
		"previous_hash",
		difficulty,
	)

	MineBlockConcurrent(&b, difficulty, 4)

	target := strings.Repeat("0", difficulty)

	if !strings.HasPrefix(b.Hash, target) {
		t.Errorf("concurrent mining did not satisfy difficulty %d, got hash %s", difficulty, b.Hash)
	}
}

func TestMineBlockConcurrentNonceReproducesHash(t *testing.T) {

	difficulty := 3

	b := block.NewBlock(
		1,
		[]ledger.Transaction{},
		"previous_hash",
		difficulty,
	)

	MineBlockConcurrent(&b, difficulty, 4)

	recalculated := b.CalculateHash()

	if recalculated != b.Hash {
		t.Errorf("stored hash does not match recalculation\nstored: %s\nrecalculated: %s", b.Hash, recalculated)
	}
}

func TestMineBlockConcurrentSingleWorkerMatchesSequential(t *testing.T) {

	difficulty := 3

	b := block.NewBlock(1, []ledger.Transaction{}, "previous_hash", difficulty)
	MineBlockConcurrent(&b, difficulty, 1)

	target := strings.Repeat("0", difficulty)
	if !strings.HasPrefix(b.Hash, target) {
		t.Error("single-worker concurrent mining should still satisfy the difficulty target")
	}
}
