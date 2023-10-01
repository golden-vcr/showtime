package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleView(res http.ResponseWriter, req *http.Request) {
	// Look up the current tape ID, defaulting to "" if no tape change has ever been
	// recorded
	tapeId, err := s.q.GetCurrentTapeId(req.Context())
	if err == sql.ErrNoRows {
		tapeId = ""
	} else if err != nil {
		fmt.Printf("Error getting tape ID: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with our current state
	state := &State{
		TapeId: tapeId,
	}
	if err := json.NewEncoder(res).Encode(state); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
