package health

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golden-vcr/showtime"
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/nicklaw5/helix/v2"
)

type GetEventsStatusFunc func() (error, error)
type GetChatStatusFunc func() error

type Server struct {
	getEventsStatus GetEventsStatusFunc
	getChatStatus   GetChatStatusFunc
}

func NewServer(client *helix.Client, channelUserId string, twitchWebhookCallbackUrl string, getChatStatus GetChatStatusFunc) *Server {
	return &Server{
		getEventsStatus: func() (error, error) {
			return events.VerifySubscriptionStatus(
				client,
				showtime.RequiredSubscriptions,
				channelUserId,
				twitchWebhookCallbackUrl,
			)
		},
		getChatStatus: getChatStatus,
	}
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	status := s.resolveStatus()
	if err := json.NewEncoder(res).Encode(status); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) resolveStatus() Status {
	err, secondaryErr := s.getEventsStatus()
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

	if err := s.getChatStatus(); err != nil {
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
