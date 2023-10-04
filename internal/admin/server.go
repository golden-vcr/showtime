package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golden-vcr/auth"
)

type Server struct{}

func (s *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// Parse claims that were resolved by our auth client's middleware
	claims, err := auth.GetClaims(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return dummy data as JSON so we know authorization succeeded
	data := struct {
		ClassifiedSecret  string            `json:"classifiedSecret"`
		AuthenticatedUser *auth.UserDetails `json:"authenticatedUser"`
	}{
		ClassifiedSecret:  "foobar",
		AuthenticatedUser: claims.User,
	}
	if err := json.NewEncoder(res).Encode(data); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func parseAuthorizationHeader(value string) string {
	prefix := "Bearer "
	if strings.HasPrefix(value, prefix) {
		return value[len(prefix):]
	}
	return value
}
