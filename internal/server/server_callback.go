package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nicklaw5/helix/v2"
)

type callbackPayload struct {
	Subscription helix.EventSubSubscription `json:"subscription"`
	Challenge    string                     `json:"challenge"`
	Event        json.RawMessage            `json:"event"`
}

func (s *Server) handlePostCallback(res http.ResponseWriter, req *http.Request) {
	// Only handle POST requests
	if req.Method != http.MethodPost {
		fmt.Printf("Got callback with unsupported method %s\n", req.Method)
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Pre-emptively read the request body so we can verify its signature
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("Failed to read body from callback request: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	// Verify that this event comes from Twitch: abort if phony
	if !helix.VerifyEventSubNotification(s.webhookSecret, req.Header, string(body)) {
		fmt.Printf("Failed to verify signature from callback request\n")
		http.Error(res, "Signature verification failed", http.StatusBadRequest)
		return
	}

	// Decode the payload as JSON so we can examine the details of the event
	var payload callbackPayload
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
		fmt.Printf("Failed to decode callback request body from JSON\n")
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// Parse the payload to a supported Event struct: if the subscription type is
	// unsupported, abort
	ev, err := parseEvent(&payload.Subscription, payload.Event)
	if err != nil {
		fmt.Printf("Failed to parse event for subscription type '%s': %v\n", payload.Subscription.Type, err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
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
	fmt.Printf("- %v\n", ev)
	res.WriteHeader(http.StatusOK)

	// If this event should produce an alert, fan that alert out to all SSE connections
	alert, err := ev.ToAlert()
	if err != nil {
		fmt.Printf("Failed to produce an alert from event of type '%s': %v\n", ev.Type, err)
		return
	}
	if alert != nil {
		s.broadcastAlert(alert)
	}
}
