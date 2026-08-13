package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"toyblockchain/block"
	"toyblockchain/chain"
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

	
	dataFlag := flag.String(
		"data",
		"",
		"blockchain persistence file (default: node_data/chain_<addr>.json)",
	)

	keyFlag := flag.String(
		"keyfile",
		"",
		"node identity key file (default: node_data/identity_<addr>.json)",
	)

	persistInterval := flag.Duration(
		"persist-interval",
		10*time.Second,
		"how often this node saves its chain to disk",
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

	dataFile := *dataFlag
	if dataFile == "" {
		dataFile = filepath.Join("node_data", fmt.Sprintf("chain_%s.json", sanitizeAddr(*addr)))
	}

	keyFile := *keyFlag
	if keyFile == "" {
		keyFile = filepath.Join("node_data", fmt.Sprintf("identity_%s.json", sanitizeAddr(*addr)))
	}

	if dir := filepath.Dir(dataFile); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("failed to create data directory: %v", err)
		}
	}

	// FR-11: load (or create, on first run) this node's identity key
	// pair, so it keeps the same identity across restarts.
	identity, err := node.LoadOrCreateIdentity(keyFile)
	if err != nil {
		log.Fatalf("failed to load node identity: %v", err)
	}

	log.Printf(
		"node identity: %s (key file: %s)",
		identity.PublicKeyHex(),
		keyFile,
	)

	
	bc, err := chain.LoadFromFile(dataFile)
	if err != nil {
		log.Fatalf("failed to load blockchain: %v", err)
	}

	log.Printf(
		"loaded chain: %d block(s) from %s",
		len(bc.Blocks),
		dataFile,
	)

	// Create the blockchain node, then swap in the loaded chain
	// before anything else touches it.
	n := node.NewNode(*addr, peers)
	n.Blockchain = bc

	// Create the HTTP server using the node package's
	// centralized route and handler implementation.
	server := node.NewServer(n, *addr)

	
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

		if err := n.SaveChainTo(dataFile); err != nil {
			log.Printf("failed to save chain on shutdown: %v", err)
		} else {
			log.Printf("chain saved to %s before shutdown", dataFile)
		}

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

	
	go n.StartPeerHealthLoop(*peerHealthInterval, stopMining)

	
	go func() {
		ticker := time.NewTicker(*persistInterval)
		defer ticker.Stop()

		for {
			select {

			case <-stopMining:
				return

			case <-ticker.C:
				if err := n.SaveChainTo(dataFile); err != nil {
					log.Printf("periodic chain save failed: %v", err)
				}
			}
		}
	}()

	if err := <-serverErrCh; err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Println("node stopped")
			return
		}

		log.Fatalf("server error: %v", err)
	}
}


func sanitizeAddr(addr string) string {
	replacer := strings.NewReplacer(":", "_", "/", "_")
	return replacer.Replace(addr)
}
