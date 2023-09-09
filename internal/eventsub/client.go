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

	requiredSubscriptions []Subscription
}

func NewClient(channelName string, clientId string, clientSecret string, webhookCallbackUrl string, webhookSecret string, requiredSubscriptions []Subscription) (*Client, error) {
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
		Client:                hc,
		channelName:           channelName,
		channelUserId:         channelUserId,
		webhookCallbackUrl:    webhookCallbackUrl,
		webhookSecret:         webhookSecret,
		requiredSubscriptions: requiredSubscriptions,
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

func (c *Client) ReconcileRequiredSubscriptions(owned []helix.EventSubSubscription) (*ReconcileResult, error) {
	params := ConditionParams{
		ChannelUserId: c.channelUserId,
	}

	requiredSubscriptionsByExistingId := make(map[string]Subscription)
	requiredSubscriptionsThatDoNotExist := make([]Subscription, 0)
	for i, requiredSubscription := range c.requiredSubscriptions {
		requiredCondition, err := params.format(&requiredSubscription.Condition)
		if err != nil {
			return nil, fmt.Errorf("failed to format condition for required subscription at index %d: %w", i, err)
		}
		subscription := findMatchingSubscription(owned, requiredSubscription.Type, requiredSubscription.Version, requiredCondition)
		if subscription != nil {
			requiredSubscriptionsByExistingId[subscription.ID] = requiredSubscription
		} else {
			requiredSubscriptionsThatDoNotExist = append(requiredSubscriptionsThatDoNotExist, requiredSubscription)
		}
	}

	relevantSubscriptions := make([]ExistingSubscription, 0)
	irrelevantSubscriptions := make([]helix.EventSubSubscription, 0)
	for i := range owned {
		requiredSubscription, isRelevant := requiredSubscriptionsByExistingId[owned[i].ID]
		if isRelevant {
			relevantSubscriptions = append(relevantSubscriptions, ExistingSubscription{
				Value:    owned[i],
				Required: requiredSubscription,
			})
		} else {
			irrelevantSubscriptions = append(irrelevantSubscriptions, owned[i])
		}
	}

	return &ReconcileResult{
		ToDelete: irrelevantSubscriptions,
		ToCreate: requiredSubscriptionsThatDoNotExist,
		Existing: relevantSubscriptions,
	}, nil
}

func (c *Client) CreateSubscription(required Subscription) error {
	params := ConditionParams{
		ChannelUserId: c.channelUserId,
	}
	requiredCondition, err := params.format(&required.Condition)
	if err != nil {
		return fmt.Errorf("failed to format condition for required '%s' subscription: %v", required.Type, err)
	}

	r, err := c.CreateEventSubSubscription(&helix.EventSubSubscription{
		Type:      required.Type,
		Version:   required.Version,
		Condition: *requiredCondition,
		Transport: helix.EventSubTransport{
			Method:   "webhook",
			Callback: c.webhookCallbackUrl,
			Secret:   c.webhookSecret,
		},
	})
	if err != nil {
		return err
	}
	if r.StatusCode != 202 {
		return fmt.Errorf("got response %d from create subscriptions request: %s", r.StatusCode, r.ErrorMessage)
	}
	return nil
}

func (c *Client) DeleteSubscription(subscriptionId string) error {
	r, err := c.RemoveEventSubSubscription(subscriptionId)
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return fmt.Errorf("got response %d from delete subscriptions request: %s", r.StatusCode, r.ErrorMessage)
	}
	return nil
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

func findMatchingSubscription(subscriptions []helix.EventSubSubscription, requiredType string, requiredVersion string, requiredCondition *helix.EventSubCondition) *helix.EventSubSubscription {
	for i := range subscriptions {
		subscription := &subscriptions[i]
		if subscription.Type != requiredType {
			continue
		}
		if subscription.Version != requiredVersion {
			continue
		}
		if !areConditionsEqual(&subscription.Condition, requiredCondition) {
			continue
		}
		return subscription
	}
	return nil
}

func areConditionsEqual(lhs *helix.EventSubCondition, rhs *helix.EventSubCondition) bool {
	return (lhs.BroadcasterUserID == rhs.BroadcasterUserID &&
		lhs.FromBroadcasterUserID == rhs.FromBroadcasterUserID &&
		lhs.ModeratorUserID == rhs.ModeratorUserID &&
		lhs.ToBroadcasterUserID == rhs.ToBroadcasterUserID &&
		lhs.RewardID == rhs.RewardID &&
		lhs.ClientID == rhs.ClientID &&
		lhs.ExtensionClientID == rhs.ExtensionClientID &&
		lhs.UserID == rhs.UserID)
}
