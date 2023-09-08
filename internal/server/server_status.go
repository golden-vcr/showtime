package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleStatus(res http.ResponseWriter, req *http.Request) {
	nyi := Status{
		IsReady: false,
		Message: "Twitch EventSub API integration has not yet been fully implemented. Twitch Events will not be received.",
	}
	if err := json.NewEncoder(res).Encode(nyi); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
