package node

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"toyblockchain/ledger"
)

// FR-3 acceptance criterion: a transaction submitted to node A
// appears in node B's pending pool.

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

// FR-3: gossip de-duplication holds across the network, not just
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

// FR-4 acceptance criterion: node A mines and broadcasts a block,
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

// FR-5 acceptance criterion: a new node with only the genesis block
// syncs the full chain from an existing peer.

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

// FR-6 acceptance criterion, and the research report's required fork
// experiment: two nodes each mine a different block at the same
// height, then reconnect. Both must converge on the same (longer)
// chain, and the orphaned block's still-valid transaction must
// return to the pending pool.

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

	// txA was only in the now-orphaned block A mined. Alice's balance
	// on B's winning chain is untouched (B's blocks carry no
	// transactions), so txA is still valid and must return to A's
	// pending pool rather than being silently dropped.
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
