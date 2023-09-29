package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) handleAuthLogin(res http.ResponseWriter, req *http.Request) {
	// Only handle POST requests
	if req.Method != http.MethodPost {
		fmt.Printf("Got login request with unsupported method %s\n", req.Method)
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	// Require a Twitch authorization code in the URL params
	code := req.URL.Query().Get("code")
	if code == "" {
		fmt.Printf("Got login request with missing 'code' parameter")
		http.Error(res, "'code' URL parameter is required", http.StatusBadRequest)
		return
	}

	// TODO: Exchange code for access token
	state := &AuthState{
		LoggedIn: false,
		Error:    "Login not implemented",
	}
	if code == "fake-code-for-testing" {
		state = &AuthState{
			LoggedIn: true,
			User: &UserDetails{
				Id:          "8675309",
				Login:       "fakeuserfortesting",
				DisplayName: "FakeUserForTesting",
			},
			Tokens: &UserTokens{
				AccessToken:  "fake-access-token-for-testing",
				RefreshToken: "fake-refresh-token-for-testing",
				Scopes:       []string{"user:read:follows", "user:read:subscriptions"},
			},
		}
	} else {
		res.WriteHeader(http.StatusUnauthorized)
	}
	if err := json.NewEncoder(res).Encode(state); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAuthRefresh(res http.ResponseWriter, req *http.Request) {
	// Only handle POST requests
	if req.Method != http.MethodPost {
		fmt.Printf("Got refresh request with unsupported method %s\n", req.Method)
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	// Require a Twitch refresh token in the Authorization header
	refreshToken := parseAuthorizationHeader(req.Header.Get("authorization"))
	if refreshToken == "" {
		fmt.Printf("Got refresh request with missing authorization header")
		http.Error(res, "Twitch refresh token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	// TODO: Use refresh token to get new access token
	state := &AuthState{
		LoggedIn: false,
		Error:    "Refresh not implemented",
	}
	if refreshToken == "fake-refresh-token-for-testing" {
		state = &AuthState{
			LoggedIn: true,
			User: &UserDetails{
				Id:          "8675309",
				Login:       "fakeuserfortesting",
				DisplayName: "FakeUserForTesting",
			},
			Tokens: &UserTokens{
				AccessToken:  "fake-access-token-for-testing",
				RefreshToken: "fake-refresh-token-for-testing",
				Scopes:       []string{"user:read:follows", "user:read:subscriptions"},
			},
		}
	} else {
		res.WriteHeader(http.StatusUnauthorized)
	}
	if err := json.NewEncoder(res).Encode(state); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAuthLogout(res http.ResponseWriter, req *http.Request) {
	// Only handle POST requests
	if req.Method != http.MethodPost {
		fmt.Printf("Got logout request with unsupported method %s\n", req.Method)
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	// Require a Twitch user access token in the Authorization header
	userAccessToken := parseAuthorizationHeader(req.Header.Get("authorization"))
	if userAccessToken == "" {
		fmt.Printf("Got logout request with missing authorization header")
		http.Error(res, "Twitch user access token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	// TODO: Revoke access token
	state := &AuthState{
		LoggedIn: false,
		Error:    "Logout not implemented",
	}
	if userAccessToken == "fake-access-token-for-testing" {
		state.Error = ""
	} else {
		res.WriteHeader(http.StatusUnauthorized)
	}
	if err := json.NewEncoder(res).Encode(state); err != nil {
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
