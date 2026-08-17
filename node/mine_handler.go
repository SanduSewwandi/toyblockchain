package node

import (
	"net/http"
)

func (s *Server) handleMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	minedBlock, ok, err := s.Node.MineOnce()

	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"mined":  false,
			"reason": "nothing to mine (empty pending pool)",
		})
		return
	}

	// Broadcast the newly mined block to peers, same as the automatic
	// mining loop does.
	s.BroadcastMinedBlock(minedBlock)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"mined": true,
		"block": minedBlock,
	})
}
