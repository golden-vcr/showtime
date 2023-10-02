package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

func (s *Server) handleGetAccess(res http.ResponseWriter, req *http.Request) {
	userAccessToken := parseAuthorizationHeader(req.Header.Get("authorization"))
	if userAccessToken == "" {
		http.Error(res, "Twitch user access token must be supplied in Authorization header", http.StatusBadRequest)
		return
	}

	claims, err := s.checkAccess(req.Context(), userAccessToken)
	if err != nil {
		if errors.Is(err, ErrTwitchReturnedUnauthorized) {
			http.Error(res, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(res).Encode(claims); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) checkAccess(ctx context.Context, accessToken string) (*AccessClaims, error) {
	user, err := s.client.ResolveUserDetailsFromAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	role := RoleViewer
	if user.ID == s.channelUserId {
		role = RoleBroadcaster
	}
	return &AccessClaims{
		User: &UserDetails{
			Id:          user.ID,
			Login:       user.Login,
			DisplayName: user.DisplayName,
		},
		Role: role,
	}, nil
}
