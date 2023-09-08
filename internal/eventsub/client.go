package eventsub

import (
	"fmt"

	"github.com/nicklaw5/helix/v2"
)

type Client struct {
	*helix.Client

	channelName        string
	channelUserId      string
	webhookCallbackUrl string
	webhookSecret      string
}

func NewClient(channelName string, clientId string, clientSecret string, webhookCallbackUrl string, webhookSecret string) (*Client, error) {
	// Create a Twitch API client, which our eventsub.Client will wrap
	hc, err := helix.NewClient(&helix.Options{
		ClientID:     clientId,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Twitch API client: %w", err)
	}

	// Obtain an application access token
	tokenResponse, err := hc.RequestAppAccessToken(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get app access token from Twitch API: %w", err)
	}
	hc.SetAppAccessToken(tokenResponse.Data.AccessToken)

	// Resolve the user ID for the channel
	channelUserId, err := getChannelUserId(hc, channelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID for channel '%s': %w", channelName, err)
	}

	return &Client{
		Client:             hc,
		channelName:        channelName,
		channelUserId:      channelUserId,
		webhookCallbackUrl: webhookCallbackUrl,
		webhookSecret:      webhookSecret,
	}, nil
}

func (c *Client) GetOwnedSubscriptions() ([]helix.EventSubSubscription, error) {
	subscriptions := make([]helix.EventSubSubscription, 0)
	params := &helix.EventSubSubscriptionsParams{
		UserID: c.channelUserId,
	}
	for {
		// Query the Twitch API for a list of our EventSub subscriptions
		r, err := c.GetEventSubSubscriptions(params)
		if err != nil {
			return nil, err
		}
		if r.StatusCode != 200 {
			return nil, fmt.Errorf("got response %d from get subscriptions request: %s", r.StatusCode, r.ErrorMessage)
		}

		for i := range r.Data.EventSubSubscriptions {
			// Ignore any subscriptions that don't hit our webhook API
			subscription := r.Data.EventSubSubscriptions[i]
			if subscription.Transport.Method != "webhook" {
				continue
			}
			if subscription.Transport.Callback != c.webhookCallbackUrl {
				continue
			}
			subscriptions = append(subscriptions, subscription)
		}

		// Continue making requests until we've seen all subscriptions
		if r.Data.Pagination.Cursor == "" {
			break
		}
		params.After = r.Data.Pagination.Cursor
	}
	return subscriptions, nil
}

func getChannelUserId(client *helix.Client, channelName string) (string, error) {
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
