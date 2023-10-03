package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Server(t *testing.T) {
	tests := []struct {
		name               string
		eventsErr          error
		eventsSecondaryErr error
		chatErr            error
		wantStatus         int
		wantIsReady        bool
		wantMessageSubstr  string
	}{
		{
			"returns 200 with isReady if no systems report errors",
			nil,
			nil,
			nil,
			http.StatusOK,
			true,
			"fully operational",
		},
		{
			"returns 200 with !isReady if events are not healthy",
			fmt.Errorf("This error is presented directly to the user"),
			nil,
			nil,
			http.StatusOK,
			false,
			"This error is presented directly to the user",
		},
		{
			"secondary error from events is surfaced in user-facing message",
			fmt.Errorf("This error is presented directly to the user"),
			fmt.Errorf("and so is this one"),
			nil,
			http.StatusOK,
			false,
			"This error is presented directly to the user (Error: and so is this one)",
		},
		{
			"returns 200 with !isReady if chat isn't ready",
			nil,
			nil,
			fmt.Errorf("mock chat error"),
			http.StatusOK,
			false,
			"chat functionality is degraded. (Error: mock chat error)",
		},
	}
	for _, tt := range tests {
		s := &Server{
			getEventsStatus: func() (error, error) {
				return tt.eventsErr, tt.eventsSecondaryErr
			},
			getChatStatus: func() error {
				return tt.chatErr
			},
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		s.ServeHTTP(res, req)

		r := res.Result()
		assert.Equal(t, tt.wantStatus, r.StatusCode)

		var status Status
		err := json.NewDecoder(r.Body).Decode(&status)
		assert.NoError(t, err)
		assert.Equal(t, tt.wantIsReady, status.IsReady)
		assert.Contains(t, status.Message, tt.wantMessageSubstr)
	}
}
