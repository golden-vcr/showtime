package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/gorilla/mux"
	"github.com/nicklaw5/helix/v2"
)

type Server struct {
	client TwitchClient
	q      Queries
}

func NewServer(twitchClientId string, twitchClientSecret string, q Queries) *Server {
	return &Server{
		client: NewTwitchClient(twitchClientId, twitchClientSecret),
		q:      q,
	}
}

func (s *Server) RegisterRoutes(r *mux.Router) {
	r.Path("/login").Methods("POST").HandlerFunc(s.handleLogin)
	r.Path("/refresh").Methods("POST").HandlerFunc(s.handleRefresh)
	r.Path("/logout").Methods("POST").HandlerFunc(s.handleLogout)
}

func (s *Server) handleLogin(res http.ResponseWriter, req *http.Request) {
	// Require a Twitch authorization code and redirect URI in the URL params
	code := req.URL.Query().Get("code")
	if code == "" {
		http.Error(res, "'code' URL parameter is required", http.StatusBadRequest)
		return
	}
	redirectUri := req.URL.Query().Get("redirect_uri")
	if redirectUri == "" {
		http.Error(res, "'redirect_uri' URL parameter is required", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for a user access token and refresh token, then
	// use our access token to resolve the full details for the auth'd user
	twitchCredentials, err := s.client.GetUserAccessToken(req.Context(), code, redirectUri)
	if err != nil {
		respondToAuthFailure(res, err)
		return
	}
	twitchUser, err := s.client.ResolveUserDetailsFromAccessToken(req.Context(), twitchCredentials.AccessToken)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record the fact that the user has logged in
	err = s.q.RecordViewerLogin(req.Context(), queries.RecordViewerLoginParams{
		TwitchUserID:      twitchUser.ID,
		TwitchDisplayName: twitchUser.DisplayName,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithLoggedIn(res, twitchUser, twitchCredentials)
}

func (s *Server) handleRefresh(res http.ResponseWriter, req *http.Request) {
	// Require a Twitch refresh token in the Authorization header
	refreshToken := parseAuthorizationHeader(req.Header.Get("authorization"))
	if refreshToken == "" {
		http.Error(res, "Twitch refresh token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	// Use that refresh token to get new access token, then look up the authenticated
	// user's details via the Twitch API
	twitchCredentials, err := s.client.RefreshUserAccessToken(req.Context(), refreshToken)
	if err != nil {
		respondToAuthFailure(res, err)
		return
	}
	twitchUser, err := s.client.ResolveUserDetailsFromAccessToken(req.Context(), twitchCredentials.AccessToken)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record the fact that the user has logged in: doing this on refresh ensures that
	// we'll periodically update our viewer.last_logged_in_at timestamp in response to
	// normal website usage
	err = s.q.RecordViewerLogin(req.Context(), queries.RecordViewerLoginParams{
		TwitchUserID:      twitchUser.ID,
		TwitchDisplayName: twitchUser.DisplayName,
	})
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithLoggedIn(res, twitchUser, twitchCredentials)
}

func (s *Server) handleLogout(res http.ResponseWriter, req *http.Request) {
	// Require a Twitch user access token in the Authorization header
	userAccessToken := parseAuthorizationHeader(req.Header.Get("authorization"))
	if userAccessToken == "" {
		http.Error(res, "Twitch user access token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	// Revoke access token, then unconditionally return a logged-out AuthState,
	// signalling errors via HTTP status code if revocation failed
	err := s.client.RevokeUserAccessToken(req.Context(), userAccessToken)
	if err != nil {
		respondToAuthFailure(res, err)
		return
	}
	respondWithLoggedOut(res, "", http.StatusOK)
}

func parseAuthorizationHeader(value string) string {
	prefix := "Bearer "
	if strings.HasPrefix(value, prefix) {
		return value[len(prefix):]
	}
	return value
}

func respondToAuthFailure(res http.ResponseWriter, err error) {
	if errors.Is(err, ErrFailedToInitializeTwitchClient) {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	respondWithLoggedOut(res, err.Error(), http.StatusUnauthorized)
}

func respondWithLoggedOut(res http.ResponseWriter, errorMessage string, status int) {
	state := &AuthState{
		LoggedIn: false,
		Error:    errorMessage,
	}
	data, marshalErr := json.Marshal(state)
	if marshalErr != nil {
		http.Error(res, marshalErr.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(status)
	res.Write(data)
}

func respondWithLoggedIn(res http.ResponseWriter, twitchUser *helix.User, twitchCredentials *helix.AccessCredentials) {
	state := &AuthState{
		LoggedIn: true,
		User: &UserDetails{
			Id:          twitchUser.ID,
			Login:       twitchUser.Login,
			DisplayName: twitchUser.DisplayName,
		},
		Tokens: &UserTokens{
			AccessToken:  twitchCredentials.AccessToken,
			RefreshToken: twitchCredentials.RefreshToken,
			Scopes:       twitchCredentials.Scopes,
		},
	}
	if err := json.NewEncoder(res).Encode(state); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}
