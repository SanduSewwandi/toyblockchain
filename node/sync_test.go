package node

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"toyblockchain/ledger"
)

func TestTransactionGossipPropagatesToPeer(t *testing.T) {

	nodes, _, testServers := startCluster(t, 2)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	resp := postTransaction(t, testServers[0].URL, tx)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted from node A, got %d", resp.StatusCode)
	}

	// gossipTransaction runs synchronously inside the handler before
	// it responds, so by the time postTransaction returns, node B
	// has already received and processed the forwarded transaction.
	if nodes[1].PendingCount() != 1 {
		t.Fatalf("expected transaction to propagate to peer B's pending pool, got %d", nodes[1].PendingCount())
	}

	if nodes[0].PendingCount() != 1 {
		t.Fatalf("expected node A's own pending pool to contain 1 transaction, got %d", nodes[0].PendingCount())
	}
}

//  gossip de-duplication holds across the network, not just
// within a single node's AddTransaction call.

func TestTransactionGossipDeduplicatesAcrossNetwork(t *testing.T) {

	nodes, _, testServers := startCluster(t, 2)

	tx := newSignedTx(t, "Alice", "Bob", 10)

	resp1 := postTransaction(t, testServers[0].URL, tx)
	resp1.Body.Close()

	if resp1.StatusCode != http.StatusAccepted {
		t.Fatalf("expected first submission to be accepted, got %d", resp1.StatusCode)
	}

	// Submit the identical transaction to node A again, simulating a
	// repeated gossip message looping back around the network.
	resp2 := postTransaction(t, testServers[0].URL, tx)
	resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected duplicate submission to return 200 OK (not re-accepted), got %d", resp2.StatusCode)
	}

	if nodes[0].PendingCount() != 1 {
		t.Fatalf("expected node A's pending pool to still contain exactly 1 transaction, got %d", nodes[0].PendingCount())
	}

	if nodes[1].PendingCount() != 1 {
		t.Fatalf("expected node B's pending pool to still contain exactly 1 transaction, got %d", nodes[1].PendingCount())
	}
}

// acceptance criterion: node A mines and broadcasts a block,
// node B validates it and its height increases by one.

func TestBlockGossipPropagatesToPeer(t *testing.T) {

	nodes, _, testServers := startCluster(t, 2)

	b := mineNextBlock(t, nodes[0], nil)

	resp := postBlock(t, testServers[0].URL, b)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted from node A, got %d", resp.StatusCode)
	}

	if nodes[0].Height() != 1 {
		t.Fatalf("expected node A's height to increase to 1, got %d", nodes[0].Height())
	}

	if nodes[1].Height() != 1 {
		t.Fatalf("expected node B's height to increase to 1 via gossip, got %d", nodes[1].Height())
	}

	if nodes[0].HeadHash() != nodes[1].HeadHash() {
		t.Fatal("expected both nodes to agree on the head hash after block gossip")
	}
}

//  GET /height returns the peer's current chain height, the first
// step of incremental synchronisation.

func TestFetchPeerHeightReturnsCurrentHeight(t *testing.T) {

	existing := NewNode("", nil)

	for i := 0; i < 3; i++ {
		b := mineNextBlock(t, existing, nil)

		if result, err := existing.AddBlock(b); !result.Accepted || err != nil {
			t.Fatalf("setup: failed to build chain: result=%+v err=%v", result, err)
		}
	}

	srv := NewServer(existing, "")
	ts := newRunningTestServer(t, srv)
	existing.Address = ts.Listener.Addr().String()

	height, err := FetchPeerHeight(existing.Address)

	if err != nil {
		t.Fatalf("unexpected error fetching peer height: %v", err)
	}

	if height != existing.Height() {
		t.Fatalf("expected fetched height %d to match peer's actual height %d", height, existing.Height())
	}
}

//  GET /blocks/{index} serves a single block by index, and returns
// 404 for an index that doesn't exist yet.

func TestFetchPeerBlockReturnsRequestedBlock(t *testing.T) {

	existing := NewNode("", nil)

	b := mineNextBlock(t, existing, nil)

	if result, err := existing.AddBlock(b); !result.Accepted || err != nil {
		t.Fatalf("setup: failed to add block: result=%+v err=%v", result, err)
	}

	srv := NewServer(existing, "")
	ts := newRunningTestServer(t, srv)
	existing.Address = ts.Listener.Addr().String()

	fetched, err := FetchPeerBlock(existing.Address, 1)

	if err != nil {
		t.Fatalf("unexpected error fetching block 1: %v", err)
	}

	if fetched.Hash != b.Hash {
		t.Fatalf("expected fetched block hash %s to match original %s", fetched.Hash, b.Hash)
	}

	if _, err := FetchPeerBlock(existing.Address, 99); err == nil {
		t.Fatal("expected an error fetching a block index that doesn't exist")
	}
}

func TestSyncFromPeerCatchesUpNewNode(t *testing.T) {

	existing := NewNode("", nil)

	for i := 0; i < 2; i++ {
		b := mineNextBlock(t, existing, nil)

		if result, err := existing.AddBlock(b); !result.Accepted || err != nil {
			t.Fatalf("setup: failed to build existing chain: result=%+v err=%v", result, err)
		}
	}

	srv := NewServer(existing, "")
	ts := newRunningTestServer(t, srv)
	existing.Address = ts.Listener.Addr().String()

	fresh := NewNode("", []string{existing.Address})

	result, err := fresh.SyncFromPeer(existing.Address)

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !result.Adopted {
		t.Fatalf("expected fresh node to adopt the peer's chain, got reason: %s", result.Reason)
	}

	if fresh.Height() != existing.Height() {
		t.Fatalf("expected heights to match after sync, got fresh=%d existing=%d", fresh.Height(), existing.Height())
	}

	if fresh.HeadHash() != existing.HeadHash() {
		t.Fatal("expected head hashes to match after sync")
	}
}

func TestForkConvergenceAndOrphanTransactionRecovery(t *testing.T) {

	nodeA := NewNode("", nil)
	nodeB := NewNode("", nil)

	// A mines a block containing a transaction. If A's fork loses,
	// this transaction becomes orphaned and must come back to life
	// in A's pending pool.
	txA := newSignedTx(t, "Alice", "Bob", 10)

	if accepted, err := nodeA.AddTransaction(txA); !accepted || err != nil {
		t.Fatalf("setup: expected txA to be accepted, got accepted=%v err=%v", accepted, err)
	}

	blockA := mineNextBlock(t, nodeA, []ledger.Transaction{txA})

	if result, err := nodeA.AddBlock(blockA); !result.Accepted || err != nil {
		t.Fatalf("setup: expected A's block to be accepted, got result=%+v err=%v", result, err)
	}

	// B mines a different block at the same height, then extends its
	// own chain by one further block so B's fork is strictly longer
	// and wins under the longest-valid-chain rule.
	blockB1 := mineNextBlock(t, nodeB, nil)

	if result, err := nodeB.AddBlock(blockB1); !result.Accepted || err != nil {
		t.Fatalf("setup: expected B's first block to be accepted, got result=%+v err=%v", result, err)
	}

	blockB2 := mineNextBlock(t, nodeB, nil)

	if result, err := nodeB.AddBlock(blockB2); !result.Accepted || err != nil {
		t.Fatalf("setup: expected B's second block to be accepted, got result=%+v err=%v", result, err)
	}

	// Reconnect: expose B over HTTP and have A sync from it.
	srvB := NewServer(nodeB, "")
	tsB := newRunningTestServer(t, srvB)
	nodeB.Address = tsB.Listener.Addr().String()

	result, err := nodeA.SyncFromPeer(nodeB.Address)

	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	if !result.Adopted {
		t.Fatalf("expected A to adopt B's longer chain, got reason: %s", result.Reason)
	}

	if nodeA.Height() != nodeB.Height() {
		t.Fatalf("expected both nodes to converge on the same height, got A=%d B=%d", nodeA.Height(), nodeB.Height())
	}

	if nodeA.HeadHash() != nodeB.HeadHash() {
		t.Fatal("expected both nodes to converge on the same head hash")
	}

	if nodeA.PendingCount() != 1 {
		t.Fatalf("expected the orphaned transaction to return to the pending pool, got %d pending", nodeA.PendingCount())
	}
}

// newRunningTestServer starts an httptest server backed by srv's
// handler and registers cleanup.
func newRunningTestServer(t *testing.T, srv *Server) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(srv.httpServer.Handler)
	t.Cleanup(ts.Close)

	return ts
}
