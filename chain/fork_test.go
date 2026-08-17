package chain

import (
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
	"toyblockchain/wallet"
)

func createChainWithBlocksAtDifficulty(
	t *testing.T,
	base *Blockchain,
	additions int,
	difficulty int,
) *Blockchain {

	t.Helper()

	bc := &Blockchain{
		Blocks: append([]block.Block{}, base.Blocks...),
	}

	w := wallet.NewWallet("test_wallet.json")

	aliceKeys, err := w.GetOrCreate("Alice")
	if err != nil {
		t.Fatalf("failed creating wallet: %v", err)
	}

	tx := ledger.Transaction{
		Sender:   "Alice",
		Receiver: "Bob",
		Amount:   1,
	}

	ledger.SignTransaction(&tx, aliceKeys)

	for i := 0; i < additions; i++ {

		if err := bc.AddBlock(
			[]ledger.Transaction{tx},
			difficulty,
		); err != nil {
			t.Fatalf("failed adding block: %v", err)
		}
	}

	return bc
}

// createChainWithBlocks keeps the old call shape (`blocks` = final total
// length, at DefaultDifficulty) for the tests that don't care about a
// specific difficulty.
func createChainWithBlocks(t *testing.T, base *Blockchain, blocks int) *Blockchain {
	t.Helper()
	return createChainWithBlocksAtDifficulty(t, base, blocks-1, DefaultDifficulty)
}

func TestResolveForkAcceptsLongerChainAtEqualDifficulty(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(t, base, 3)
	candidate := createChainWithBlocks(t, base, 5)

	accepted, msg := current.ResolveFork(candidate)
	t.Log(msg)

	if !accepted {
		t.Fatal("expected the longer, higher-work chain to be accepted")
	}

	if len(current.Blocks) != 5 {
		t.Fatalf("expected 5 blocks, got %d", len(current.Blocks))
	}
}

func TestResolveForkRejectsShorterChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(t, base, 5)
	candidate := createChainWithBlocks(t, base, 3)

	accepted, msg := current.ResolveFork(candidate)
	t.Log(msg)

	if accepted {
		t.Fatal("expected the shorter, lower-work chain to be rejected")
	}

	if len(current.Blocks) != 5 {
		t.Fatal("current chain changed")
	}
}

// This is the actual "heaviest chain" behavior the stretch goal asks
// for: a candidate with FEWER blocks but MORE cumulative proof-of-work
// must win over a longer, easier-to-mine chain.
func TestResolveForkAcceptsHeavierChainDespiteFewerBlocks(t *testing.T) {

	base := NewBlockchain()

	// 3 easy blocks: total added work = 3 * 16^3 = 12,288
	current := createChainWithBlocksAtDifficulty(t, base, 3, MinDifficulty)

	// 1 harder block: added work = 16^4 = 65,536 — well over 5x current's
	candidate := createChainWithBlocksAtDifficulty(t, base, 1, MinDifficulty+1)

	if len(candidate.Blocks) >= len(current.Blocks) {
		t.Fatalf("test setup invalid: candidate must have fewer blocks than current")
	}

	accepted, msg := current.ResolveFork(candidate)
	t.Log(msg)

	if !accepted {
		t.Fatal("expected the shorter but heavier chain to be accepted")
	}

	if len(current.Blocks) != len(candidate.Blocks) {
		t.Fatal("expected current chain to be replaced by the heavier candidate")
	}
}

// The inverse: more blocks but less cumulative work must lose.
func TestResolveForkRejectsLighterChainDespiteMoreBlocks(t *testing.T) {

	base := NewBlockchain()

	// 1 hard block for current: work = 16^4 = 65,536
	current := createChainWithBlocksAtDifficulty(t, base, 1, MinDifficulty+1)

	// 3 easy blocks for candidate: work = 3 * 16^3 = 12,288 — more
	// blocks, far less total work.
	candidate := createChainWithBlocksAtDifficulty(t, base, 3, MinDifficulty)

	accepted, msg := current.ResolveFork(candidate)
	t.Log(msg)

	if accepted {
		t.Fatal("expected the longer but lighter chain to be rejected")
	}

	if len(current.Blocks) != 2 { // genesis + 1 added block
		t.Fatal("current chain changed")
	}
}

func TestResolveForkTieBreaksOnFinalBlockHashWhenWorkIsEqual(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(t, base, 4)
	candidate := createChainWithBlocks(t, base, 4)

	currentWork := ChainWork(current)
	candidateWork := ChainWork(candidate)

	if currentWork.Cmp(candidateWork) != 0 {
		t.Fatalf(
			"test setup invalid: expected equal work, got current=%s candidate=%s",
			currentWork, candidateWork,
		)
	}

	expectAccept := candidate.Blocks[len(candidate.Blocks)-1].Hash >
		current.Blocks[len(current.Blocks)-1].Hash

	accepted, msg := current.ResolveFork(candidate)
	t.Log(msg)

	if accepted != expectAccept {
		t.Fatalf(
			"tie-break outcome mismatch: expected accepted=%v, got %v",
			expectAccept, accepted,
		)
	}
}

func TestResolveForkRejectsInvalidChain(t *testing.T) {

	base := NewBlockchain()

	current := createChainWithBlocks(t, base, 3)
	candidate := createChainWithBlocks(t, base, 4)

	// Tamper candidate
	candidate.Blocks[2].PreviousHash = "invalid"

	accepted, msg := current.ResolveFork(candidate)
	t.Log(msg)

	if accepted {
		t.Fatal("expected invalid chain rejected")
	}

	if len(current.Blocks) != 3 {
		t.Fatal("current chain changed")
	}
}
