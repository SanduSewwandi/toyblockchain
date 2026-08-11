package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"toyblockchain/node"
)

func main() {

	addr := flag.String("addr", ":8001", "address for this node to listen on")
	peersFlag := flag.String("peers", "", "comma-separated list of peer addresses")

	flag.Parse()

	var peers []string

	if *peersFlag != "" {
		peers = strings.Split(*peersFlag, ",")
	}

	n := node.NewNode(*addr, peers)

	mux := http.NewServeMux()

	mux.HandleFunc("/height", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"height":%d}`, n.Height())
	})

	mux.HandleFunc("/peers", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"peers":%q}`, n.PeerList())
	})

	log.Printf("node listening on %s, peers=%v\n", *addr, peers)
	log.Fatal(http.ListenAndServe(*addr, mux))
}