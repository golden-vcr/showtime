package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golden-vcr/showtime"
	"github.com/golden-vcr/showtime/internal/events"
)

func (s *Server) handleStatus(res http.ResponseWriter, req *http.Request) {
	status := s.resolveStatus()
	if err := json.NewEncoder(res).Encode(status); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) resolveStatus() Status {
	err, secondaryErr := events.VerifySubscriptionStatus(s.twitchClient, showtime.RequiredSubscriptions, s.channelUserId, s.twitchConfig.WebhookCallbackUrl)
	if err != nil {
		suffix := ""
		if secondaryErr != nil {
			suffix = fmt.Sprintf(" (Error: %s)", secondaryErr.Error())
		}
		return Status{
			IsReady: false,
			Message: err.Error() + suffix,
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
