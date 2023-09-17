package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nicklaw5/helix/v2"
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

	reconciled, err := s.eventsub.ReconcileRequiredSubscriptions(subscriptions)
	if err != nil {
		return Status{
			IsReady: false,
			Message: fmt.Sprintf(
				"Unable to ascertain status of required Twitch event subscriptions. This may indicate a problem with the Golden VCR server. (Error: %s)",
				err.Error(),
			),
		}
	}
	if len(reconciled.ToCreate) > 0 {
		return Status{
			IsReady: false,
			Message: "One or more required Twitch event subscriptions do not yet exist. The Golden VCR server may not be receiving all required data from Twitch.",
		}
	}
	for _, existing := range reconciled.Existing {
		if existing.Value.Status != helix.EventSubStatusEnabled {
			return Status{
				IsReady: false,
				Message: "One or more required Twitch event subscriptions are disabled. The Golden VCR server may not be receiving all required data from Twitch.",
			}
		}
	}

	// TODO: Maybe don't commingle notification status and chat status?
	if err := s.chat.GetStatus(); err != nil {
		return Status{
			IsReady: false,
			Message: fmt.Sprintf(
				"Twitch Event subscriptions are fully operational, but chat functionality is degraded. (Error: %s)",
				err,
			),
		}
	}

	return Status{
		IsReady: true,
		Message: "All required Twitch Event subscriptions are enabled, and chat features are working. The Golden VCR server is fully operational!",
	}
}
