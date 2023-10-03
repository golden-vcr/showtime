package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nicklaw5/helix/v2"
)

type VerifyNotificationFunc func(header http.Header, message string) bool
type HandleEventFunc func(ctx context.Context, subscription *helix.EventSubSubscription, data json.RawMessage) error

type Server struct {
	verifyNotification VerifyNotificationFunc
	handleEvent        HandleEventFunc
}

func NewServer(twitchWebhookSecret string, eventHandler *Handler) *Server {
	return &Server{
		verifyNotification: func(header http.Header, message string) bool {
			return helix.VerifyEventSubNotification(twitchWebhookSecret, header, message)
		},
		handleEvent: eventHandler.HandleEvent,
	}
}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	s.handlePostCallback(res, req)
}

func (s *Server) handlePostCallback(res http.ResponseWriter, req *http.Request) {
	// Pre-emptively read the request body so we can verify its signature
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("Failed to read body from callback request: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	// Verify that this event comes from Twitch: abort if phony
	if !s.verifyNotification(req.Header, string(body)) {
		fmt.Printf("Failed to verify signature from callback request\n")
		http.Error(res, "Signature verification failed", http.StatusBadRequest)
		return
	}

	// Decode the payload from JSON so we can examine the details of the event
	var payload struct {
		Subscription helix.EventSubSubscription `json:"subscription"`
		Challenge    string                     `json:"challenge"`
		Event        json.RawMessage            `json:"event"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
		fmt.Printf("Failed to decode callback request body from JSON\n")
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// If the challenge value is set, Twitch is sending us an initial request to
	// confirm registration of this event callback: responding with the same value will
	// enable the event subscription. This occurs after the parseEvent check so that we
	// won't allow subscriptions to be created until we fully support the relevant
	// event type.
	if payload.Challenge != "" {
		fmt.Printf("Responding to challenge with %q\n", payload.Challenge)
		res.Write([]byte(payload.Challenge))
		return
	}

	// We can accept the event, so respond with 200
	fmt.Printf("Got event of type %q\n", payload.Subscription.Type)
	fmt.Printf("- %s\n", string(payload.Event))
	res.WriteHeader(http.StatusOK)

	if err := s.handleEvent(req.Context(), &payload.Subscription, payload.Event); err != nil {
		fmt.Printf("Failed to handle event of type '%s': %v\n", payload.Subscription.Type, err)
	}
}
