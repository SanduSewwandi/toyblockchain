package node

import (
	"testing"
)

// FR-10: a node learns about a peer's peers, not just the peers it
// was configured with.
func TestExchangePeersDiscoversNewPeer(t *testing.T) {

	nodes, _, testServers := startCluster(t, 2)

	// Give node A a third peer address that B doesn't know about yet,
	// then have B exchange with A and pick it up.
	nodes[0].mu.Lock()
	nodes[0].Peers["localhost:9999"] = true
	nodes[0].mu.Unlock()

	addrA := testServers[0].Listener.Addr().String()

	if err := nodes[1].ExchangePeersWith(addrA); err != nil {
		t.Fatalf("unexpected error exchanging peers: %v", err)
	}

	found := false

	for _, p := range nodes[1].PeerList() {
		if p == "localhost:9999" {
			found = true
		}
	}

	if !found {
		t.Fatal("expected node B to discover localhost:9999 via peer exchange with A")
	}
}

// FR-10: a peer that fails health checks maxPeerFailures times in a
// row is dropped from the peer set.
func TestRunPeerHealthCheckDropsUnreachablePeer(t *testing.T) {

	n := NewNode("", []string{"localhost:1"}) // port 1 is never a real listener

	failures := make(map[string]int)

	for i := 0; i < maxPeerFailures; i++ {
		n.RunPeerHealthCheck(failures)
	}

	for _, p := range n.PeerList() {
		if p == "localhost:1" {
			t.Fatal("expected unreachable peer to be dropped after repeated failures")
		}
	}
}

// FR-10: a peer that responds successfully is never dropped, however
// many health-check passes run.
func TestRunPeerHealthCheckKeepsHealthyPeer(t *testing.T) {

	nodes, _, testServers := startCluster(t, 2)

	addrB := testServers[1].Listener.Addr().String()

	failures := make(map[string]int)

	for i := 0; i < maxPeerFailures+2; i++ {
		nodes[0].RunPeerHealthCheck(failures)
	}

	found := false

	for _, p := range nodes[0].PeerList() {
		if p == addrB {
			found = true
		}
	}

	if !found {
		t.Fatal("expected healthy peer to remain in the peer set")
	}
}
