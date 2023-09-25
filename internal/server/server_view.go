package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleView(res http.ResponseWriter, req *http.Request) {
	// Handle a preflight OPTIONS request: we want our Twitch extension (hosted at
	// 'https://<client-id>.twitch-ext.tv') to be able to use this endpoint, so we need
	// to send back the appropriate CORS headers to instruct the browser that it's safe
	// for a page served by that origin to send asynchronous requests to this API
	if req.Method == http.MethodOptions {
		// TEMP: Just dump headers and return 405
		fmt.Printf("OPTIONS request!\n")
		for k, v := range req.Header {
			fmt.Printf("- %s: %s\n", k, v)
		}
		fmt.Printf("Preflight handler NYI; returning 405\n")
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	// Only handle GET requests
	if req.Method != http.MethodGet {
		fmt.Printf("Got view request with unsupported method %s\n", req.Method)
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

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
