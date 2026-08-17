package node

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const maxPeerFailures = 3

var peerHealthHTTPClient = &http.Client{
	Timeout: 3 * time.Second,
}

// peersResponse mirrors the JSON shape returned by handlePeers in
// server.go: {"peers": ["addr1", "addr2", ...]}.
type peersResponse struct {
	Peers []string `json:"peers"`
}

// ExchangePeersWith fetches peer's known peer list over /peers and
// merges any new addresses into this node's own peer set (skipping
// itself and anything already known).
func (n *Node) ExchangePeersWith(peer string) error {

	url := "http://" + peer + "/peers"

	resp, err := peerHealthHTTPClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var body peersResponse

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	for _, addr := range body.Peers {

		if addr == "" || addr == n.Address {
			continue
		}

		if !n.Peers[addr] {
			n.Peers[addr] = true
			log.Printf("discovered new peer %s via %s", addr, peer)
		}
	}

	// The peer that told us about these addresses is, by definition,
	// reachable — make sure it's in our own set too.
	if peer != "" && peer != n.Address {
		n.Peers[peer] = true
	}

	return nil
}

func checkPeerHealth(peer string) bool {

	resp, err := peerHealthHTTPClient.Get("http://" + peer + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (n *Node) RunPeerHealthCheck(failures map[string]int) {

	peers := n.PeerList()

	for _, peer := range peers {

		if checkPeerHealth(peer) {

			failures[peer] = 0

			if err := n.ExchangePeersWith(peer); err != nil {
				log.Printf("peer exchange with %s failed: %v", peer, err)
			}

			continue
		}

		failures[peer]++

		log.Printf(
			"peer %s unreachable (%d/%d)",
			peer,
			failures[peer],
			maxPeerFailures,
		)

		if failures[peer] >= maxPeerFailures {

			n.mu.Lock()
			delete(n.Peers, peer)
			n.mu.Unlock()

			delete(failures, peer)

			log.Printf(
				"dropped peer %s after %d consecutive failed health checks",
				peer,
				maxPeerFailures,
			)
		}
	}
}

// StartPeerHealthLoop runs RunPeerHealthCheck on a timer until stop
// is closed. Intended to run alongside StartMiningLoop in cmd/node.
func (n *Node) StartPeerHealthLoop(interval time.Duration, stop <-chan struct{}) {

	failures := make(map[string]int)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {

		case <-stop:
			return

		case <-ticker.C:
			n.RunPeerHealthCheck(failures)
		}
	}
}
