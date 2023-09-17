package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleAlerts(res http.ResponseWriter, req *http.Request) {
	// Only handle GET requests
	if req.Method != http.MethodGet {
		fmt.Printf("Got alerts request with unsupported method %s\n", req.Method)
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

	// Open a channel to receive alert notifications as they happen
	ch := make(chan *Alert, 32)
	handle := s.subscribeToAlerts(ch)

	fmt.Printf("Sending live alert notifications to %s...\n", req.RemoteAddr)
	for {
		select {
		case <-time.After(30 * time.Second):
			fmt.Printf(">> sending keepalive\n")
			res.Write([]byte(":\n\n"))
			res.(http.Flusher).Flush()
		case alert := <-ch:
			data, err := json.Marshal(alert)
			if err != nil {
				fmt.Printf("Failed to serialize alert of type '%s' as JSON: %v\n", alert.Type, err)
				continue
			}
			fmt.Printf(">> sending data: %s\n", data)
			fmt.Fprintf(res, "data: %s\n\n", data)
			res.(http.Flusher).Flush()
		case <-req.Context().Done():
			fmt.Printf("Stopping live alert notifications to %s.\n", req.RemoteAddr)
			s.unsubscribeFromAlerts(handle)
			return
		}
	}
}
