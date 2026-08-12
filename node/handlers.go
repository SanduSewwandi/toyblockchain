package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"toyblockchain/block"
	"toyblockchain/ledger"
)

const originPeerHeader = "X-Origin-Peer"

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	var tx ledger.Transaction

	if err := decodeJSONBody(w, r, &tx); err != nil {
		return
	}

	fromPeer := r.Header.Get(originPeerHeader)

	accepted, err := s.Node.AddTransaction(tx)
	if err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	// Duplicate transactions are harmless. AddTransaction returns
	// accepted=false with nil error when the transaction was already seen.
	if !accepted {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"accepted": false,
			"reason":   "transaction already seen",
		})
		return
	}

	// Gossip only newly accepted transactions.
	gossipErrors := s.gossipTransaction(tx, fromPeer)

	response := map[string]interface{}{
		"accepted": true,
		"reason":   "transaction accepted",
	}

	if len(gossipErrors) > 0 {
		response["gossip_errors"] = gossipErrors
	}

	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) gossipTransaction(tx ledger.Transaction, skipFrom string) []string {
	peers := s.Node.PeerList()

	var errors []string

	for _, peer := range peers {
		if peer == "" || peer == s.Node.Address || peer == skipFrom {
			continue
		}

		if err := sendTransactionToPeer(peer, tx, s.Node.Address); err != nil {
			errors = append(
				errors,
				fmt.Sprintf("%s: %v", peer, err),
			)
		}
	}

	return errors
}

func sendTransactionToPeer(peer string, tx ledger.Transaction, fromAddress string) error {
	return postJSON(peer, "/transactions", tx, fromAddress)
}

func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	var b block.Block

	if err := decodeJSONBody(w, r, &b); err != nil {
		return
	}

	fromPeer := r.Header.Get(originPeerHeader)

	result, err := s.Node.AddBlock(b)

	if err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	if !result.Accepted {
		// A duplicate is harmless and should not be treated as a
		// server failure.
		if result.Reason == "block already seen" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"accepted": false,
				"reason":   result.Reason,
			})
			return
		}

		if result.Reason == "block does not extend current chain" {

			go s.Node.HandleUnexpectedBlock(b, fromPeer)
		}

		writeJSON(w, http.StatusConflict, map[string]interface{}{
			"accepted": false,
			"reason":   result.Reason,
		})
		return
	}

	// Gossip only accepted blocks.
	gossipErrors := s.gossipBlock(b, fromPeer)

	response := map[string]interface{}{
		"accepted": true,
		"reason":   result.Reason,
	}

	if len(gossipErrors) > 0 {
		response["gossip_errors"] = gossipErrors
	}

	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) gossipBlock(b block.Block, skipFrom string) []string {
	peers := s.Node.PeerList()

	var errors []string

	for _, peer := range peers {
		if peer == "" || peer == s.Node.Address || peer == skipFrom {
			continue
		}

		if err := sendBlockToPeer(peer, b, s.Node.Address); err != nil {
			errors = append(
				errors,
				fmt.Sprintf("%s: %v", peer, err),
			)
		}
	}

	return errors
}

func sendBlockToPeer(peer string, b block.Block, fromAddress string) error {
	return postJSON(peer, "/blocks", b, fromAddress)
}

// BroadcastMinedBlock gossips a block this node just mined to its peers.
// Unlike handleBlocks, there's no origin peer to skip — the block
// originated here, not from a gossip message.
func (s *Server) BroadcastMinedBlock(b block.Block) {
	if errs := s.gossipBlock(b, ""); len(errs) > 0 {
		log.Printf(
			"gossip errors broadcasting mined block %d: %v",
			b.Index,
			errs,
		)
	}
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	const maxBodySize = 1 << 20 // 1 MiB

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	contentType := r.Header.Get("Content-Type")

	if contentType != "" &&
		!strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		writeJSONError(
			w,
			http.StatusUnsupportedMediaType,
			"Content-Type must be application/json",
		)
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(dst); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			fmt.Sprintf("invalid JSON body: %v", err),
		)
		return fmt.Errorf("failed to decode JSON body: %w", err)
	}

	// Reject a second JSON value in the same request body.
	var extra interface{}

	if err := decoder.Decode(&extra); err == nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"request body must contain exactly one JSON value",
		)
		return fmt.Errorf("request contains multiple JSON values")
	}

	return nil
}

// Peer HTTP helper
var peerHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

func postJSON(peer string, path string, value interface{}, fromAddress string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	url := fmt.Sprintf("http://%s%s", peer, path)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(originPeerHeader, fromAddress)

	resp, err := peerHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("peer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("peer returned HTTP status %d", resp.StatusCode)
	}

	return nil
}
