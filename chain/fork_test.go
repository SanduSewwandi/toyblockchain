package chain

import (
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
	"toyblockchain/wallet"
)


func createChainWithDifficulties(
	t *testing.T,
	base *Blockchain,
	difficulties []int,
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

	for _, difficulty := range difficulties {

		if err := bc.AddBlock(
			[]ledger.Transaction{tx},
			difficulty,
		); err != nil {

			t.Fatalf(
				"failed adding block at difficulty %d: %v",
				difficulty,
				err,
			)
		}
	}

	return bc
}

// This is the exact scenario from the stretch-goal spec: same height,
// but chain B's second block is mined at a higher difficulty, so B has
// more cumulative work even though neither chain has more blocks.
func TestResolveForkAcceptsHeavierChainAtSameHeight(t *testing.T) {

	base := NewBlockchain()

	chainA := createChainWithDifficulties(t, base, []int{4, 4})
	chainB := createChainWithDifficulties(t, base, []int{4, 6})

	if len(chainA.Blocks) != len(chainB.Blocks) {
		t.Fatalf(
			"expected equal height chains, got %d vs %d",
			len(chainA.Blocks),
			len(chainB.Blocks),
		)
	}

	if ChainWork(chainA).Cmp(ChainWork(chainB)) >= 0 {
		t.Fatalf(
			"expected chain B to have more work than chain A, got A=%s B=%s",
			ChainWork(chainA).String(),
			ChainWork(chainB).String(),
		)
	}

	accepted, msg := chainA.ResolveFork(chainB)

	t.Log(msg)

	if !accepted {
		t.Fatal(
			"expected heavier chain B to be accepted despite equal height",
		)
	}

	if len(chainA.Blocks) != len(chainB.Blocks) {
		t.Fatal(
			"expected current chain to adopt candidate's block count",
		)
	}
}

// Mirror of the above: the lighter chain must never win, regardless of
// which side of the call it's on.
func TestResolveForkRejectsLighterChainAtSameHeight(t *testing.T) {

	base := NewBlockchain()

	heavier := createChainWithDifficulties(t, base, []int{4, 6})
	lighter := createChainWithDifficulties(t, base, []int{4, 4})

	accepted, msg := heavier.ResolveFork(lighter)

	t.Log(msg)

	if accepted {
		t.Fatal(
			"expected lighter chain to be rejected",
		)
	}

	if len(heavier.Blocks) != 3 {
		t.Fatal(
			"current chain changed after rejecting a lighter candidate",
		)
	}
}

// The core of the "heaviest chain, not longest chain" requirement: a
// candidate with strictly more blocks must still lose if its total
// work is lower than the current chain's.
func TestResolveForkRejectsMoreBlocksButLessWork(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithDifficulties(t, base, []int{6, 6})
	candidate := createChainWithDifficulties(t, base, []int{3, 3, 3})

	if len(candidate.Blocks) <= len(current.Blocks) {
		t.Fatalf(
			"test setup error: expected candidate to have more blocks than current, got %d vs %d",
			len(candidate.Blocks),
			len(current.Blocks),
		)
	}

	if ChainWork(candidate).Cmp(ChainWork(current)) >= 0 {
		t.Fatalf(
			"test setup error: expected candidate to have less work than current, got candidate=%s current=%s",
			ChainWork(candidate).String(),
			ChainWork(current).String(),
		)
	}

	accepted, msg := current.ResolveFork(candidate)

	t.Log(msg)

	if accepted {
		t.Fatal(
			"expected a longer but lighter chain to be rejected",
		)
	}

	if len(current.Blocks) != 3 {
		t.Fatal(
			"current chain changed after rejecting a longer-but-lighter candidate",
		)
	}
}

// When two chains have exactly equal cumulative work, the winner must
// be decided deterministically by the final block hash — not by which
// side happens to call ResolveFork first.
func TestResolveForkTieBreaksOnFinalBlockHash(t *testing.T) {

	base := NewBlockchain()

	chainA := createChainWithDifficulties(t, base, []int{4, 4})
	chainB := createChainWithDifficulties(t, base, []int{4, 4})

	if ChainWork(chainA).Cmp(ChainWork(chainB)) != 0 {
		t.Fatalf(
			"test setup error: expected equal work for a tie-break test, got A=%s B=%s",
			ChainWork(chainA).String(),
			ChainWork(chainB).String(),
		)
	}

	headA := chainA.Blocks[len(chainA.Blocks)-1].Hash
	headB := chainB.Blocks[len(chainB.Blocks)-1].Hash

	if headA == headB {
		t.Skip("mined identical head hashes by coincidence; cannot test tie-break")
	}

	aShouldWin := headA < headB

	// Case 1: A is current, B is the candidate.
	current1 := &Blockchain{Blocks: append([]block.Block{}, chainA.Blocks...)}
	accepted1, msg1 := current1.ResolveFork(chainB)
	t.Log(msg1)

	if aShouldWin && accepted1 {
		t.Fatal("expected chain A (smaller final hash) to keep current, rejecting B")
	}

	if !aShouldWin && !accepted1 {
		t.Fatal("expected chain B (smaller final hash) to be adopted over A")
	}

	// Case 2: B is current, A is the candidate — must reach the same
	// absolute conclusion regardless of which side initiates.
	current2 := &Blockchain{Blocks: append([]block.Block{}, chainB.Blocks...)}
	accepted2, msg2 := current2.ResolveFork(chainA)
	t.Log(msg2)

	if aShouldWin && !accepted2 {
		t.Fatal("expected chain A (smaller final hash) to be adopted over B")
	}

	if !aShouldWin && accepted2 {
		t.Fatal("expected chain B (smaller final hash) to keep current, rejecting A")
	}
}

func TestResolveForkRejectsInvalidChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithDifficulties(t, base, []int{4, 4})
	candidate := createChainWithDifficulties(t, base, []int{4, 4, 4})

	// Tamper candidate so it has more blocks AND more raw work, but is
	// structurally invalid — it must still be rejected outright,
	// before work is ever compared.
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