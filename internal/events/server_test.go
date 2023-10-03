package events

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nicklaw5/helix/v2"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handlePostCallback(t *testing.T) {
	tests := []struct {
		name                 string
		requestBody          string
		signatureIsOK        bool
		wantStatus           int
		wantBody             string
		wantHandledEventData string
	}{
		{
			"if signature verification fails, returns 400",
			"{}",
			false,
			http.StatusBadRequest,
			"Signature verification failed",
			"",
		},
		{
			"if challenge is set, echoes challenge with 200",
			`{"subscription":{"id":"some-subscription"},"challenge":"foobar12345"}`,
			true,
			http.StatusOK,
			"foobar12345",
			"",
		},
		{
			"valid event is recorded via handle func",
			`{"subscription":{"id":"some-subscription","type":"test"},"event":{"value":42}}`,
			true,
			http.StatusOK,
			"",
			`{"value":42}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handledEventData := ""
			s := &Server{
				verifyNotification: func(header http.Header, message string) bool {
					return tt.signatureIsOK
				},
				handleEvent: func(ctx context.Context, subscription *helix.EventSubSubscription, data json.RawMessage) error {
					handledEventData = string(data)
					return nil
				},
			}
			req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(tt.requestBody))
			res := httptest.NewRecorder()
			s.handlePostCallback(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)

			assert.Equal(t, tt.wantHandledEventData, handledEventData)
		})
	}
}
