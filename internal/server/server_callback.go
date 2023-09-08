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
	if req.Method != http.MethodPost {
		fmt.Printf("Got callback with unsupported method %s\n", req.Method)
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("Failed to read body from callback request: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer req.Body.Close()

	if !helix.VerifyEventSubNotification(s.webhookSecret, req.Header, string(body)) {
		fmt.Printf("Failed to verify signature from callback request\n")
		http.Error(res, "Signature verification failed", http.StatusBadRequest)
		return
	}

	var payload callbackPayload
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
		fmt.Printf("Failed to decode callback request body from JSON\n")
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if payload.Challenge != "" {
		fmt.Printf("Responding to challenge with %q", payload.Challenge)
		res.Write([]byte(payload.Challenge))
		return
	}

	fmt.Printf("Got event of subscription of type %q\n", payload.Subscription.Type)
	res.WriteHeader(http.StatusOK)
}
