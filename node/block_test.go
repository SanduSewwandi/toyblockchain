package node

import (
	"toyblockchain/block"
	"toyblockchain/chain"
	"toyblockchain/ledger"

	"testing"
)

// FR-4: a valid block that extends the tip is accepted.

func TestAddBlockAcceptsValidExtendingBlock(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)
	b := mineNextBlock(t, n, []ledger.Transaction{tx})

	result, err := n.AddBlock(b)

	if err != nil {
		t.Fatalf("expected valid block to be accepted, got error: %v", err)
	}

	if !result.Accepted {
		t.Fatalf("expected block to be accepted, got reason: %s", result.Reason)
	}

	if n.Height() != 1 {
		t.Fatalf("expected height 1 after accepting a block, got %d", n.Height())
	}
}

// FR-4 + FR-9: an accepted block's transactions are removed from the
// pending pool.

func TestAddBlockRemovesMinedTransactionFromPending(t *testing.T) {

	n := NewNode(":0", nil)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	if accepted, err := n.AddTransaction(tx); !accepted || err != nil {
		t.Fatalf("setup: expected transaction to be accepted, got accepted=%v err=%v", accepted, err)
	}

	b := mineNextBlock(t, n, []ledger.Transaction{tx})

	if result, err := n.AddBlock(b); !result.Accepted || err != nil {
		t.Fatalf("expected block to be accepted, got result=%+v err=%v", result, err)
	}

	if n.PendingCount() != 0 {
		t.Fatalf("expected mined transaction to be cleared from pending pool, got %d remaining", n.PendingCount())
	}
}

// Regression test for the seenBlocks-marked-too-early bug: a block
// that fails to extend the tip must NOT be marked seen, so a later
// sync/reorg attempt can still process it (rather than silently
// dropping it as a duplicate).

func TestAddBlockDoesNotMarkNonExtendingBlockAsSeen(t *testing.T) {

	n := NewNode(":0", nil)

	// Build a block that is well-formed (valid hash/PoW/Merkle root)
	// but does not extend the current chain: wrong index, unrelated
	// previous hash.
	stale := block.NewBlock(5, nil, "0000nonexistentprevioushash", chain.MinDifficulty)
	chain.MineBlock(&stale, chain.MinDifficulty)

	result1, err1 := n.AddBlock(stale)

	if result1.Accepted {
		t.Fatal("expected non-extending block to be rejected")
	}

	if err1 != nil {
		t.Fatalf("expected no error for a well-formed but non-extending block, got: %v", err1)
	}

	if result1.Reason != "block does not extend current chain" {
		t.Fatalf("expected reason 'block does not extend current chain', got: %s", result1.Reason)
	}

	// Submitting the identical block again must be reprocessed, not
	// short-circuited as "block already seen" — that would make
	// legitimate reorg/sync recovery impossible.
	result2, _ := n.AddBlock(stale)

	if result2.Reason == "block already seen" {
		t.Fatal("non-extending block must remain re-processable, not marked as seen")
	}

	if result2.Reason != "block does not extend current chain" {
		t.Fatalf("expected the block to be re-evaluated identically the second time, got reason: %s", result2.Reason)
	}
}

// An accepted block IS marked seen, so gossip loops don't reprocess
// it endlessly (FR-3/FR-4 de-duplication).

func TestAddBlockDeduplicatesAcceptedBlock(t *testing.T) {

	n := NewNode(":0", nil)

	b := mineNextBlock(t, n, nil)

	result1, err1 := n.AddBlock(b)

	if !result1.Accepted || err1 != nil {
		t.Fatalf("expected block to be accepted, got result=%+v err=%v", result1, err1)
	}

	result2, err2 := n.AddBlock(b)

	if result2.Accepted {
		t.Fatal("expected the same block submitted again to be rejected")
	}

	if err2 != nil {
		t.Fatalf("expected duplicate block rejection to be silent (nil error), got: %v", err2)
	}

	if result2.Reason != "block already seen" {
		t.Fatalf("expected reason 'block already seen', got: %s", result2.Reason)
	}

	if n.Height() != 1 {
		t.Fatalf("expected height to remain 1 after a duplicate submission, got %d", n.Height())
	}
}

// Regression test for the missing difficulty-retargeting check: a
// block that satisfies its own (too-low) stated difficulty must
// still be rejected if it doesn't match what the chain actually
// expects at this position.

func TestAddBlockRejectsBlockBelowExpectedDifficulty(t *testing.T) {

	n := NewNode(":0", nil)

	n.mu.RLock()
	latest := n.Blockchain.GetLatestBlock()
	expected := chain.NextDifficultyFor(n.Blockchain, chain.DefaultDifficulty)
	n.mu.RUnlock()

	if expected <= chain.MinDifficulty {
		t.Skip("expected difficulty is already at the minimum; cannot construct a lower valid block")
	}

	// Mine a block that is internally self-consistent (its hash truly
	// satisfies its own stated difficulty) but at MinDifficulty,
	// below what NextDifficultyFor actually expects here.
	cheap := block.NewBlock(latest.Index+1, nil, latest.Hash, chain.MinDifficulty)
	chain.MineBlock(&cheap, chain.MinDifficulty)

	result, err := n.AddBlock(cheap)

	if result.Accepted {
		t.Fatal("expected block below the expected retargeted difficulty to be rejected")
	}

	if result.Reason != "invalid difficulty" {
		t.Fatalf("expected reason 'invalid difficulty', got: %s", result.Reason)
	}

	if err == nil {
		t.Fatal("expected an error describing the difficulty mismatch")
	}
}

// FR-4 acceptance criterion: an invalid block (tampered hash) is
// rejected outright.

func TestAddBlockRejectsTamperedHash(t *testing.T) {

	n := NewNode(":0", nil)

	b := mineNextBlock(t, n, nil)
	b.Hash = "0000tamperedhash"

	result, err := n.AddBlock(b)

	if result.Accepted {
		t.Fatal("expected block with tampered hash to be rejected")
	}

	if err == nil {
		t.Fatal("expected an error for tampered hash")
	}
}
