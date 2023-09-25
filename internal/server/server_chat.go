package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golden-vcr/showtime/internal/chat"
)

func (s *Server) handleChat(res http.ResponseWriter, req *http.Request) {
	// Only handle GET requests
	if req.Method != http.MethodGet {
		fmt.Printf("Got chat request with unsupported method %s\n", req.Method)
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	// If a content-type is explicitly requested, require that it's text/event-stream
	accept := req.Header.Get("accept")
	if accept != "" && accept != "*/*" && !strings.HasPrefix(accept, "text/event-stream") {
		fmt.Printf("Got alerts request with unsupported content-type %s\n", accept)
		http.Error(res, "unsupported content-type", http.StatusBadRequest)
		return
	}

	// Keep the connection alive and open a text/event-stream response body
	res.Header().Set("content-type", "text/event-stream")
	res.Header().Set("cache-control", "no-cache")
	res.Header().Set("connection", "keep-alive")
	res.WriteHeader(http.StatusOK)
	res.(http.Flusher).Flush()

	// Open a channel to receive chat lines as they're emitted
	ch := make(chan *chat.Event, 32)
	handle := s.chatEvents.register(ch)

	// Send an initial keepalive message: this ensures that Cloudfront will kick into
	// action immediately without requiring special configuration rules
	res.Write([]byte(":\n\n"))
	res.(http.Flusher).Flush()

	// Send all incoming chat lines to the client for as long as the connection is open
	fmt.Printf("Sending live chat messages to %s...\n", req.RemoteAddr)
	for {
		select {
		case <-time.After(30 * time.Second):
			res.Write([]byte(":\n\n"))
			res.(http.Flusher).Flush()
		case alert := <-ch:
			data, err := json.Marshal(alert)
			if err != nil {
				fmt.Printf("Failed to serialize chat line as JSON: %v\n", err)
				continue
			}
			fmt.Fprintf(res, "data: %s\n\n", data)
			res.(http.Flusher).Flush()
		case <-req.Context().Done():
			fmt.Printf("Stopping live chat notifications to %s.\n", req.RemoteAddr)
			s.chatEvents.unregister(handle)
			return
		}
	}
}
