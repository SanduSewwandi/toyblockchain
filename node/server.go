package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// Block-by-index endpoint (FR-5): serves a single block, e.g.
// GET /blocks/3. This is the per-block fetch used by incremental sync,
// distinct from the bulk chain dump above and from POST /blocks (block
// gossip), which are handled separately.
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

// Peers endpoint
func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
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
