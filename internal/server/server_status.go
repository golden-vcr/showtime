package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleStatus(res http.ResponseWriter, req *http.Request) {
	status := s.resolveStatus()
	if err := json.NewEncoder(res).Encode(status); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) resolveStatus() Status {
	subscriptions, err := s.eventsub.GetOwnedSubscriptions()
	if err != nil {
		return Status{
			IsReady: false,
			Message: fmt.Sprintf(
				"Unable to retrieve subscription details from the Twitch API."+
					" This may be due to a disruption in service from Twitch itself,"+
					" or there may be a problem with the Golden VCR server."+
					" (Error: %s)",
				err.Error(),
			),
		}
	}

	if len(subscriptions) == 0 {
		return Status{
			IsReady: false,
			Message: "No Twitch event subscriptions are enabled. The Golden VCR server may not yet be fully connected to the Golden VCR Twitch account.",
		}
	}

	return Status{
		IsReady: false,
		Message: "Twitch EventSub API integration has not yet been fully implemented. Twitch Events will not be received.",
	}
}
