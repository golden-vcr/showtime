package twitch

import (
	"fmt"
	"net/http"

	"github.com/nicklaw5/helix/v2"
)

func NewClientWithAppToken(clientId string, clientSecret string) (*helix.Client, error) {
	c, err := helix.NewClient(&helix.Options{
		ClientID:     clientId,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Twitch API client: %w", err)
	}

	res, err := c.RequestAppAccessToken(nil)
	if err == nil && res.StatusCode != http.StatusOK {
		err = fmt.Errorf("got status %d: %s", res.StatusCode, res.ErrorMessage)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get app access token from Twitch API: %w", err)
	}

	c.SetAppAccessToken(res.Data.AccessToken)
	return c, nil
}
