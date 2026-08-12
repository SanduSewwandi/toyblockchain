package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

// startCluster spins up n nodes, each with a real httptest HTTP server,
// wired so every node knows every other node's actual listen address
// (the way cmd/node's -peers flag would, but resolved automatically
// since httptest assigns ports dynamically). Servers are torn down
// automatically at the end of the test.
func startCluster(t *testing.T, n int) ([]*Node, []*Server, []*httptest.Server) {
	t.Helper()

	nodes := make([]*Node, n)
	srvs := make([]*Server, n)
	testServers := make([]*httptest.Server, n)
	addrs := make([]string, n)

	// First pass: create nodes and unstarted servers so we know each
	// one's actual address before wiring peer lists.
	for i := 0; i < n; i++ {
		nd := NewNode("", nil)
		srv := NewServer(nd, "")
		ts := httptest.NewUnstartedServer(srv.httpServer.Handler)

		nodes[i] = nd
		srvs[i] = srv
		testServers[i] = ts
		addrs[i] = ts.Listener.Addr().String()
	}

	// Second pass: wire each node's Address and full peer list, then
	// start serving.
	for i := 0; i < n; i++ {
		nodes[i].Address = addrs[i]

		peers := make(map[string]bool)

		for j, a := range addrs {
			if j != i {
				peers[a] = true
			}
		}

		nodes[i].Peers = peers

		testServers[i].Start()

		ts := testServers[i]
		t.Cleanup(ts.Close)
	}

	return nodes, srvs, testServers
}

func postTransaction(t *testing.T, baseURL string, tx ledger.Transaction) *http.Response {
	t.Helper()

	body, err := json.Marshal(tx)
	if err != nil {
		t.Fatalf("failed to marshal transaction: %v", err)
	}

	resp, err := http.Post(baseURL+"/transactions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /transactions failed: %v", err)
	}

	return resp
}

func postBlock(t *testing.T, baseURL string, b block.Block) *http.Response {
	t.Helper()

	body, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("failed to marshal block: %v", err)
	}

	resp, err := http.Post(baseURL+"/blocks", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /blocks failed: %v", err)
	}

	return resp
}
