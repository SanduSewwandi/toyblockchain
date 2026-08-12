package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"toyblockchain/block"
	"toyblockchain/node"
)

func main() {
	addr := flag.String(
		"addr",
		":8001",
		"address for this node to listen on",
	)

	peersFlag := flag.String(
		"peers",
		"",
		"comma-separated list of peer addresses",
	)

	mineInterval := flag.Duration(
		"mine-interval",
		5*time.Second,
		"how often this node attempts to mine pending transactions",
	)

	noMine := flag.Bool(
		"no-mine",
		false,
		"disable this node's mining loop (still validates and gossips)",
	)

	peerHealthInterval := flag.Duration(
		"peer-health-interval",
		10*time.Second,
		"how often this node checks peer health and exchanges peer lists",
	)

	flag.Parse()

	// Parse peer addresses.
	var peers []string

	if *peersFlag != "" {
		for _, peer := range strings.Split(*peersFlag, ",") {
			peer = strings.TrimSpace(peer)

			if peer != "" {
				peers = append(peers, peer)
			}
		}
	}

	// Create the blockchain node.
	n := node.NewNode(*addr, peers)

	// Create the HTTP server using the node package's
	// centralized route and handler implementation.
	server := node.NewServer(n, *addr)

	// Handle Ctrl+C / process termination so the HTTP server
	// and mining loop can shut down gracefully.
	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		os.Interrupt,
		syscall.SIGTERM,
	)

	stopMining := make(chan struct{})

	go func() {
		<-stop

		log.Println("shutdown signal received")

		close(stopMining)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	// Start the HTTP server in the background so it doesn't block the
	// initial sync attempt below.
	serverErrCh := make(chan error, 1)

	go func() {
		if err := server.Start(); err != nil {
			serverErrCh <- err
		}
	}()

	log.Printf(
		"node listening on %s, peers=%v",
		*addr,
		peers,
	)

	// Give the listener a brief moment to come up before syncing, so a
	// cluster of nodes started together doesn't all fail their first
	// sync attempt against a peer that isn't ready yet. Peers still
	// down are simply skipped — SyncFromPeers logs per-peer results
	// without stopping the node.
	if len(peers) > 0 {
		time.Sleep(200 * time.Millisecond)

		for _, result := range n.SyncFromPeers() {
			if result.Adopted {
				log.Printf(
					"synced chain from peer: %d -> %d blocks (%s)",
					result.LocalHeight+1,
					result.RemoteHeight+1,
					result.Reason,
				)
			}
		}
	}

	if !*noMine {
		go n.StartMiningLoop(*mineInterval, func(b block.Block) {
			server.BroadcastMinedBlock(b)
		}, stopMining)
	}

	// FR-10: peer health. Runs regardless of whether mining is
	// enabled — exchanges peer lists with known peers (so a network
	// can form from a single seed address) and drops any peer that
	// fails repeated health checks.
	go n.StartPeerHealthLoop(*peerHealthInterval, stopMining)

	if err := <-serverErrCh; err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Println("node stopped")
			return
		}

		log.Fatalf("server error: %v", err)
	}
}
