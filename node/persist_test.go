package node

import (
	"path/filepath"
	"testing"

	"toyblockchain/chain"
)

func TestSaveChainToPersistsCurrentChain(t *testing.T) {

	n := NewNode("", nil)

	b := mineNextBlock(t, n, nil)

	if result, err := n.AddBlock(b); !result.Accepted || err != nil {
		t.Fatalf("setup: expected block to be accepted, got result=%+v err=%v", result, err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "chain.json")

	if err := n.SaveChainTo(path); err != nil {
		t.Fatalf("unexpected error saving chain: %v", err)
	}

	loaded, err := chain.LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error loading saved chain: %v", err)
	}

	if len(loaded.Blocks) != n.Height()+1 {
		t.Fatalf("expected %d blocks in saved chain, got %d", n.Height()+1, len(loaded.Blocks))
	}

	if loaded.Blocks[len(loaded.Blocks)-1].Hash != n.HeadHash() {
		t.Fatal("expected saved chain's head hash to match the node's current head hash")
	}
}


func TestReloadedChainMatchesSavedChain(t *testing.T) {

	n := NewNode("", nil)

	for i := 0; i < 2; i++ {
		b := mineNextBlock(t, n, nil)

		if result, err := n.AddBlock(b); !result.Accepted || err != nil {
			t.Fatalf("setup: failed building chain: result=%+v err=%v", result, err)
		}
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "chain.json")

	if err := n.SaveChainTo(path); err != nil {
		t.Fatalf("unexpected error saving chain: %v", err)
	}

	// Simulate a restart: a brand new node, loading the previous
	// node's persisted chain instead of a fresh genesis.
	restarted := NewNode("", nil)

	loaded, err := chain.LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error reloading chain: %v", err)
	}

	restarted.Blockchain = loaded

	if restarted.Height() != n.Height() {
		t.Fatalf("expected restarted node to match original height, got %d vs %d", restarted.Height(), n.Height())
	}

	if restarted.HeadHash() != n.HeadHash() {
		t.Fatal("expected restarted node to match original head hash")
	}
}
