package node

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"toyblockchain/crypto"
	"toyblockchain/ledger"
)


func TestConcurrentMiningGossipAndReadsAreRaceFree(t *testing.T) {

	if testing.Short() {
		t.Skip("skipping concurrency stress test in -short mode")
	}

	nodes, srvs, testServers := startCluster(t, 2)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Node A mines continuously in the background and broadcasts
	// each mined block to B via the real gossip path.
	wg.Add(1)
	go func() {
		defer wg.Done()
		nodes[0].StartMiningLoop(10*time.Millisecond, srvs[0].BroadcastMinedBlock, stop)
	}()

	// A dedicated goroutine feeds Alice's real signed transactions
	// into node A's HTTP endpoint at a steady pace, staying within
	// her genesis balance of 100.
	wg.Add(1)
	go func() {
		defer wg.Done()

		aliceKey, err := crypto.GenerateKeyPair()
		if err != nil {
			return
		}

		for i := 0; i < 20; i++ {
			select {
			case <-stop:
				return
			default:
			}

			tx := ledger.Transaction{
				Sender:   "Alice",
				Receiver: "Bob",
				Amount:   1,
			}
			ledger.SignTransaction(&tx, aliceKey)

			resp := postTransaction(t, testServers[0].URL, tx)
			resp.Body.Close()

			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Several goroutines hammer garbage/duplicate/invalid
	// transactions concurrently, exercising the rejection paths
	// under contention (unregistered senders, bad signatures).
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()

			for i := 0; i < 15; i++ {
				select {
				case <-stop:
					return
				default:
				}

				kp, err := crypto.GenerateKeyPair()
				if err != nil {
					continue
				}

				tx := ledger.Transaction{
					Sender:   "NoSuchAccount",
					Receiver: "Bob",
					Amount:   1,
				}
				ledger.SignTransaction(&tx, kp)

				resp := postTransaction(t, testServers[worker%len(testServers)].URL, tx)
				resp.Body.Close()
			}
		}(w)
	}

	// Concurrent readers hit every introspection endpoint on both
	// nodes throughout the run.
	readEndpoints := []string{"/health", "/status", "/height", "/chain", "/peers", "/pending"}

	for _, ts := range testServers {
		for _, ep := range readEndpoints {
			wg.Add(1)
			go func(baseURL, endpoint string) {
				defer wg.Done()

				for {
					select {
					case <-stop:
						return
					default:
					}

					resp, err := http.Get(baseURL + endpoint)
					if err == nil {
						resp.Body.Close()
					}

					time.Sleep(3 * time.Millisecond)
				}
			}(ts.URL, ep)
		}
	}

	// Also hammer the exported read-lock methods directly (bypassing
	// HTTP) to exercise Node's RWMutex from goroutines with no
	// network hop in between.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				_ = nodes[0].Height()
				_ = nodes[0].HeadHash()
				_ = nodes[0].PeerList()
				_ = nodes[0].PendingCount()
				_ = nodes[0].Balance("Alice")
				_ = nodes[0].ChainSnapshot()
			}
		}()
	}

	// Let everything run concurrently for a short window, then stop
	// and wait for every goroutine to exit cleanly.
	time.Sleep(400 * time.Millisecond)
	close(stop)
	wg.Wait()

	// Sanity check: the node should still be in a coherent state —
	// not corrupted, not deadlocked, and able to answer a request.
	if nodes[0].Height() < 0 {
		t.Fatal("node A chain height is invalid after concurrent access")
	}
}
