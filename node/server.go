package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"toyblockchain/block"
)

// Server represents the HTTP server for a blockchain node.
type Server struct {
	Node       *Node
	Address    string
	httpServer *http.Server
}

func NewServer(n *Node, address string) *Server {
	s := &Server{
		Node:    n,
		Address: address,
	}

	mux := http.NewServeMux()

	// Node information endpoints (FR-8)

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/height", s.handleHeight)
	mux.HandleFunc("/chain", s.handleChain)
	mux.HandleFunc("/peers", s.handlePeers)
	mux.HandleFunc("/balance", s.handleBalance)
	mux.HandleFunc("/pending", s.handlePending)

	mux.HandleFunc("/transactions", s.handleTransactions)
	mux.HandleFunc("/blocks", s.handleBlocks)
	mux.HandleFunc("/mine", s.handleMine)

	// Merkle inclusion proof endpoint. Registered before the generic
	// "/blocks/" subtree route below, since Go's ServeMux resolves the
	// more specific pattern first regardless of registration order,
	// but keeping it here documents that ordering intent.
	mux.HandleFunc("GET /blocks/{index}/proof/{txIndex}", s.handleMerkleProof)

	mux.HandleFunc("/blocks/", s.handleBlockByIndex)

	s.httpServer = &http.Server{
		Addr:              address,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	if s.Node == nil {
		return fmt.Errorf("cannot start server: node is nil")
	}

	if s.Address == "" {
		return fmt.Errorf("cannot start server: address is empty")
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}

// Health endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"address":       s.Node.Address,
		"height":        s.Node.Height(),
		"head_hash":     s.Node.HeadHash(),
		"pending_count": s.Node.PendingCount(),
		"peers":         s.Node.PeerList(),
	})
}

// Height endpoint
func (s *Server) handleHeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"height": s.Node.Height(),
	})
}

// Chain endpoint
func (s *Server) handleChain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	blocks := s.Node.ChainSnapshot()

	writeJSON(w, http.StatusOK, blocks)
}

func (s *Server) handleBlockByIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	indexStr := strings.TrimPrefix(r.URL.Path, "/blocks/")

	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid block index")
		return
	}

	b, ok := s.Node.BlockAt(index)
	if !ok {
		writeJSONError(
			w,
			http.StatusNotFound,
			fmt.Sprintf("block %d not found", index),
		)
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// Merkle inclusion proof endpoint: GET /blocks/{index}/proof/{txIndex}.
// Proves a single transaction was included in a specific block without
// requiring the caller to have the block's full transaction list.
func (s *Server) handleMerkleProof(w http.ResponseWriter, r *http.Request) {

	blockIndex, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || blockIndex < 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid block index")
		return
	}

	txIndex, err := strconv.Atoi(r.PathValue("txIndex"))
	if err != nil || txIndex < 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid transaction index")
		return
	}

	b, ok := s.Node.BlockAt(blockIndex)
	if !ok {
		writeJSONError(
			w,
			http.StatusNotFound,
			fmt.Sprintf("block %d not found", blockIndex),
		)
		return
	}

	proof, err := block.GenerateMerkleProof(b.Transactions, txIndex)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, proof)
}

func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if self := r.URL.Query().Get("self"); self != "" {
		s.Node.AddPeer(self)
	}

	writeJSON(w, http.StatusOK, map[string][]string{
		"peers": s.Node.PeerList(),
	})
}

// Balance endpoint

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	address := r.URL.Query().Get("address")

	if address == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"address is required",
		)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"address": address,
		"balance": s.Node.Balance(address),
	})
}

// Pending endpoint
func (s *Server) handlePending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"pending": s.Node.PendingCount(),
	})
}

// JSON helpers
func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)

		fmt.Printf("%s %s %d %s\n", r.Method, r.URL.Path, recorder.statusCode, duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
