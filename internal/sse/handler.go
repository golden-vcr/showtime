package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Handler is an HTTP handler that serves a stream of data using Server-Sent Events
type Handler[T any] struct {
	ctx context.Context
	b   bus[T]

	OnConnectEventFunc func() T
}

// NewHandler initializes an SSE handler that will read messages from the given channel
// and fan them out to all extant HTTP connections
func NewHandler[T any](ctx context.Context, ch <-chan T) *Handler[T] {
	h := &Handler[T]{
		ctx: ctx,
		b: bus[T]{
			chs: make(map[chan T]struct{}),
		},
	}
	go func() {
		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
				h.b.clear()
			case message := <-ch:
				h.b.publish(message)
			}
		}
	}()
	return h
}

// ServeHTTP responds by opening a long-lived HTTP connection to which events will be
// written as the handler receives them, formatted as text/event-stream messages with
// 'data' consisting of a JSON-encoded message payload
func (h *Handler[T]) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// If a content-type is explicitly requested, require that it's text/event-stream
	accept := req.Header.Get("accept")
	if accept != "" && accept != "*/*" && !strings.HasPrefix(accept, "text/event-stream") {
		message := fmt.Sprintf("content-type %s is not supported", accept)
		http.Error(res, message, http.StatusBadRequest)
		return
	}

	// Keep the connection alive and open a text/event-stream response body
	res.Header().Set("content-type", "text/event-stream")
	res.Header().Set("cache-control", "no-cache")
	res.Header().Set("connection", "keep-alive")
	res.WriteHeader(http.StatusOK)
	res.(http.Flusher).Flush()

	// If configured to send an initial value immediately upon connect, resolve that
	// value and send it: otherwise send an initial keepalive message to ensure that
	// Cloudflare will kick into action immediately without requiring special
	// configuration rules
	if h.OnConnectEventFunc != nil {
		message := h.OnConnectEventFunc()
		data, err := json.Marshal(message)
		if err != nil {
			fmt.Printf("Failed to serialize SSE message as JSON: %v\n", err)
		} else {
			fmt.Fprintf(res, "data: %s\n\n", data)
			res.(http.Flusher).Flush()
		}
	} else {
		res.Write([]byte(":\n\n"))
		res.(http.Flusher).Flush()
	}

	// Open a channel to receive message structs (i.e. any JSON-serializable value that
	// we want to send over our stream) as they're emitted
	ch := make(chan T, 32)
	h.b.register(ch)

	// Send all incoming messages to the client for as long as the connection is open
	fmt.Printf("Opened SSE connection to %s...\n", req.RemoteAddr)
	for {
		select {
		case <-time.After(30 * time.Second):
			res.Write([]byte(":\n\n"))
			res.(http.Flusher).Flush()
		case message := <-ch:
			data, err := json.Marshal(message)
			if err != nil {
				fmt.Printf("Failed to serialize SSE message as JSON: %v\n", err)
				continue
			}
			fmt.Fprintf(res, "data: %s\n\n", data)
			res.(http.Flusher).Flush()
		case <-h.ctx.Done():
			fmt.Printf("Server is shutting down; abandoning SSE connection to %s.\n", req.RemoteAddr)
			h.b.unregister(ch)
			return
		case <-req.Context().Done():
			fmt.Printf("SSE connection to %s has been closed.\n", req.RemoteAddr)
			h.b.unregister(ch)
			return
		}
	}
}
