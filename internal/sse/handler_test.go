package sse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Handler(t *testing.T) {
	t.Run("server responds by opening an SSE connection", func(t *testing.T) {
		h := NewHandler[struct{}](context.Background(), make(<-chan struct{}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		go h.ServeHTTP(res, req)
		waitForResponseSubstring(t, res, ":")

		assert.Equal(t, http.StatusOK, res.Code)
		assert.Equal(t, "text/event-stream", res.Header().Get("content-type"))
		assert.Equal(t, "no-cache", res.Header().Get("cache-control"))
		assert.Equal(t, "keep-alive", res.Header().Get("connection"))
	})
	t.Run("if explict 'accept' is set, it must be 'text/event-stream'", func(t *testing.T) {
		h := NewHandler[struct{}](context.Background(), make(<-chan struct{}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("accept", "application/json")
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)

		assert.Equal(t, http.StatusBadRequest, res.Code)
	})
	t.Run("messages sent to channel are fanned out to all connected clients", func(t *testing.T) {
		// Prepare a channel into which we can write {x,y} coordinate messages, and pass
		// that into a handler that we can use to fan those messages out to each HTTP
		// client connection that's open
		coords := make(chan coordinate, 32)
		h := NewHandler[coordinate](context.Background(), coords)

		// Send an inital message into our channel: no subscribers are registered, so it
		// should just be dropped
		coords <- coordinate{100, 1}
		time.Sleep(5 * time.Millisecond)

		// Etsablish a separate child context for each of two HTTP clients
		ctxA, closeA := context.WithCancel(context.Background())
		ctxB, closeB := context.WithCancel(context.Background())
		defer closeA()
		defer closeB()

		// Initialize requests that will use those contexts
		reqA := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctxA)
		reqB := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctxB)
		resA := httptest.NewRecorder()
		resB := httptest.NewRecorder()

		// Connect client A, and while it's connected, emit a new message
		go h.ServeHTTP(resA, reqA)
		waitForResponseSubstring(t, resA, ":")
		coords <- coordinate{200, 2}
		waitForResponseSubstring(t, resA, `"x":200`)

		// Connect client B, then emit a new message which both clients should receive
		go h.ServeHTTP(resB, reqB)
		waitForResponseSubstring(t, resB, ":")
		coords <- coordinate{300, 3}
		waitForResponseSubstring(t, resA, `"x":300`)
		waitForResponseSubstring(t, resB, `"x":300`)

		// Disconnect client A, then emit a final message
		closeA()
		blockUntil(t, func() bool { return len(h.b.chs) == 1 }, 5*time.Millisecond)
		coords <- coordinate{400, 4}
		waitForResponseSubstring(t, resB, `"x":400`)

		// Verify that each client got the expected set of data in the response body
		bodyA, err := io.ReadAll(resA.Body)
		assert.NoError(t, err)
		assert.Equal(t, ":\n\ndata: {\"x\":200,\"y\":2}\n\ndata: {\"x\":300,\"y\":3}\n\n", string(bodyA))

		bodyB, err := io.ReadAll(resB.Body)
		assert.NoError(t, err)
		assert.Equal(t, ":\n\ndata: {\"x\":300,\"y\":3}\n\ndata: {\"x\":400,\"y\":4}\n\n", string(bodyB))
	})
	t.Run("canceling the handler's context closes all connections", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		coords := make(chan coordinate, 32)
		h := NewHandler[coordinate](ctx, coords)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()

		go h.ServeHTTP(res, req)
		waitForResponseSubstring(t, res, ":")
		coords <- coordinate{222, 0}
		waitForResponseSubstring(t, res, `"x":222`)

		cancel()
		blockUntil(t, func() bool { return len(h.b.chs) == 0 }, 5*time.Millisecond)
		coords <- coordinate{333, 0}

		time.Sleep(5 * time.Millisecond)
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.Equal(t, ":\n\ndata: {\"x\":222,\"y\":0}\n\n", string(body))
	})
}

type coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func waitForResponseSubstring(t *testing.T, res *httptest.ResponseRecorder, s string) {
	bodyContainsSubstring := func() bool {
		return strings.Contains(res.Body.String(), s)
	}
	blockUntil(t, bodyContainsSubstring, 5*time.Millisecond)
}

func blockUntil(t *testing.T, cond func() bool, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatal("timed out waiting for condition")
		case <-time.After(100 * time.Microsecond):
			if cond() {
				return
			}
		}
	}
}
