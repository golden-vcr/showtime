package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/nicklaw5/helix/v2"
)

func (s *Server) handleAuthLogin(res http.ResponseWriter, req *http.Request) {
	// Only handle POST requests
	if req.Method != http.MethodPost {
		fmt.Printf("Got login request with unsupported method %s\n", req.Method)
		http.Error(res, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	// Require a Twitch authorization code and redirect URI in the URL params
	code := req.URL.Query().Get("code")
	if code == "" {
		fmt.Printf("Got login request with missing 'code' parameter\n")
		http.Error(res, "'code' URL parameter is required", http.StatusBadRequest)
		return
	}
	redirectUri := req.URL.Query().Get("redirect_uri")
	if redirectUri == "" {
		fmt.Printf("Got login request with missing 'redirect_uri' parameter\n")
		http.Error(res, "'redirect_uri' URL parameter is required", http.StatusBadRequest)
		return
	}

	// Exchange code for access token
	client, err := helix.NewClientWithContext(req.Context(), &helix.Options{
		ClientID:     s.twitchConfig.ClientId,
		ClientSecret: s.twitchConfig.ClientSecret,
		RedirectURI:  redirectUri,
	})
	if err != nil {
		fmt.Printf("failed to initialize twitch client for login: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	tokenResponse, err := client.RequestUserAccessToken(code)
	if err != nil {
		fmt.Printf("failed to get user access token: %v\n", err)
		respondWithLoggedOut(res, err.Error())
		return
	}
	if tokenResponse.StatusCode != http.StatusOK {
		fmt.Printf("got %d response from RequestUserAccessToken: %s\n", tokenResponse.StatusCode, tokenResponse.ErrorMessage)
		respondWithLoggedOut(res, tokenResponse.ErrorMessage)
		return
	}

	// Resolve details for the auth'd user given our new access token
	client.SetUserAccessToken(tokenResponse.Data.AccessToken)
	twitchUser, err := resolveTwitchUser(client)
	if err != nil {
		fmt.Printf("failed to resolve twitch user from access token post-login: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Record the fact that the user has logged in
	err = s.q.RecordViewerLogin(req.Context(), queries.RecordViewerLoginParams{
		TwitchUserID:      twitchUser.ID,
		TwitchDisplayName: twitchUser.DisplayName,
	})
	if err != nil {
		fmt.Printf("failed to record user login: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithLoggedIn(res, twitchUser, &tokenResponse.Data)
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
		fmt.Printf("Got refresh request with missing authorization header\n")
		http.Error(res, "Twitch refresh token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	// Use refresh token to get new access token
	client, err := helix.NewClientWithContext(req.Context(), &helix.Options{
		ClientID:     s.twitchConfig.ClientId,
		ClientSecret: s.twitchConfig.ClientSecret,
	})
	if err != nil {
		fmt.Printf("failed to initialize twitch client for refresh: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	refreshResponse, err := client.RefreshUserAccessToken(refreshToken)
	if err != nil {
		fmt.Printf("failed to refresh access token: %v\n", err)
		respondWithLoggedOut(res, err.Error())
		return
	}
	if refreshResponse.StatusCode != http.StatusOK {
		fmt.Printf("got %d response from RefreshUserAccessToken: %s\n", refreshResponse.StatusCode, refreshResponse.ErrorMessage)
		respondWithLoggedOut(res, refreshResponse.ErrorMessage)
		return
	}

	// Resolve details for the auth'd user given our new access token
	client.SetUserAccessToken(refreshResponse.Data.AccessToken)
	twitchUser, err := resolveTwitchUser(client)
	if err != nil {
		fmt.Printf("failed to resolve twitch user from access token post-refresh: %v\n", err)
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
		fmt.Printf("failed to record user login: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithLoggedIn(res, twitchUser, &refreshResponse.Data)
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
		fmt.Printf("Got logout request with missing authorization header\n")
		http.Error(res, "Twitch user access token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	// TODO: Revoke access token
	client, err := helix.NewClientWithContext(req.Context(), &helix.Options{
		ClientID: s.twitchConfig.ClientId,
	})
	if err != nil {
		fmt.Printf("failed to initialize twitch client for logout: %v\n", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	revokeResponse, err := client.RevokeUserAccessToken(userAccessToken)
	if err != nil {
		fmt.Printf("failed to revoke user access token: %v\n", err)
		respondWithLoggedOut(res, err.Error())
		return
	}
	if revokeResponse.StatusCode != http.StatusOK {
		fmt.Printf("got %d response from RevokeUserAccessToken: %s\n", revokeResponse.StatusCode, revokeResponse.ErrorMessage)
		respondWithLoggedOut(res, revokeResponse.ErrorMessage)
		return
	}
	respondWithLoggedOut(res, "")
}

func parseAuthorizationHeader(value string) string {
	prefix := "Bearer "
	if strings.HasPrefix(value, prefix) {
		return value[len(prefix):]
	}
	return value
}

func resolveTwitchUser(client *helix.Client) (*helix.User, error) {
	r, err := client.GetUsers(&helix.UsersParams{})
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got response %d from Users: %s", r.StatusCode, r.ErrorMessage)
	}
	if len(r.Data.Users) != 1 {
		return nil, fmt.Errorf("GetUsers returned %d results; expected 1", len(r.Data.Users))
	}
	return &r.Data.Users[0], nil
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

func respondWithLoggedOut(res http.ResponseWriter, errorMessage string) {
	state := &AuthState{
		LoggedIn: false,
		Error:    errorMessage,
	}
	data, marshalErr := json.Marshal(state)
	if marshalErr != nil {
		http.Error(res, marshalErr.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusUnauthorized)
	res.Write(data)
}
