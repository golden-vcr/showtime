package twitch

import (
	"fmt"

	"github.com/nicklaw5/helix/v2"
)

func GetChannelUserId(client *helix.Client, channelName string) (string, error) {
	r, err := client.GetUsers(&helix.UsersParams{
		Logins: []string{channelName},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get user ID: %w", err)
	}
	if r.StatusCode != 200 {
		return "", fmt.Errorf("got response %d from get users request: %s", r.StatusCode, r.ErrorMessage)
	}
	if len(r.Data.Users) != 1 {
		return "", fmt.Errorf("got %d results from get users request; expected exactly 1", len(r.Data.Users))
	}
	return r.Data.Users[0].ID, nil
}
