package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/nicklaw5/helix/v2"
)

const BasePointsForSubscription = 600
const BasePointsForGiftSub = 200

// handleChannelSubscriptionEvent responds to 'channel.subscribe': "This subscription
// type sends a notification when a specified channel receives a subscriber. This does
// not include resubscribes."
//   - payload: {tier: "1000", is_gift: false, <user info>}
func (h *Handler) handleChannelSubscriptionEvent(ctx context.Context, data json.RawMessage) error {
	// Decode the JSON payload carried by the event
	var ev helix.EventSubChannelSubscribeEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelSubscribeEvent: %w", err)
	}
	fmt.Printf("Got channel.subscribe: %+v\n", ev)

	// Record the fact that the user is now subscribed
	err := h.q.RecordViewerSubscribe(ctx, queries.RecordViewerSubscribeParams{
		TwitchUserID:      ev.UserID,
		TwitchDisplayName: ev.UserName,
	})
	if err != nil {
		return fmt.Errorf("RecordViewerSubscribe failed: %w\n", err)
	}

	// If the subscription is tier 2 or 3, award 2x or 5x the points
	multiplier, err := getCreditMultiplierFromTier(ev.Tier)
	if err != nil {
		return nil
	}

	// Contact the auth server to request a short-lived JWT that will give us
	// authoritative access to the backend resources associated with the user who's now
	// a subscriber
	fmt.Printf("Requesting JWT in response to subscription for user %s\n", ev.UserName)
	accessToken, err := h.authServiceClient.RequestServiceToken(ctx, auth.ServiceTokenRequest{
		Service: "showtime",
		User: auth.UserDetails{
			Id:          ev.UserID,
			Login:       ev.UserLogin,
			DisplayName: ev.UserName,
		},
	})
	if err != nil {
		return fmt.Errorf("RequestServiceToken failed: %w", err)
	}

	// Send a request to the ledger server to request that points be granted to the user
	fmt.Printf("Requesting credit of %d points (before %.fx multiplier) to user %s\n", BasePointsForSubscription, multiplier, ev.UserName)
	_, err = h.ledgerClient.RequestCreditFromSubscription(ctx, accessToken, BasePointsForSubscription, true, ev.IsGift, "", multiplier)
	if err != nil {
		return fmt.Errorf("RequestCreditFromSubscription failed: %v", err)
	}

	// Finally, emit an alert so this user's new subscription will be acknowledged in
	// the on-stream overlay graphics
	h.alertsChan <- &alerts.Alert{
		Type: alerts.AlertTypeSubscribe,
		Data: alerts.AlertData{
			Subscribe: &alerts.AlertDataSubscribe{
				Username:            ev.UserName,
				IsGift:              ev.IsGift,
				NumCumulativeMonths: 1,
				Message:             "",
			},
		},
	}

	return nil
}

// handleChannelSubscriptionEndEvent responds to 'channel.subscription.end': "This
// subscription type sends a notification when a subscription to the specified channel
// expires."
//   - payload: {tier: "1000", is_gift: false, <user info>}
func (h *Handler) handleChannelSubscriptionEndEvent(ctx context.Context, data json.RawMessage) error {
	// Decode the JSON payload carried by the event: both 'channel.subscribe' and
	// 'channel.subscription.end' use the same payload type
	var ev helix.EventSubChannelSubscribeEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelSubscribeEvent: %w", err)
	}
	fmt.Printf("Got channel.subscription.end: %+v\n", ev)

	return nil
}

// handleChannelSubscriptionGiftEvent responds to 'channel.subscription.gift': "This
// subscription type sends a notification when a user gives one or more gifted
// subscriptions in a channel."
//   - payload: {total: 2, tier: "1000", cumulative_total?: 284, is_anonymous: false,
//     <user info>}
func (h *Handler) handleChannelSubscriptionGiftEvent(ctx context.Context, data json.RawMessage) error {
	// Decode the JSON payload carried by the event
	var ev helix.EventSubChannelSubscriptionGiftEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelSubscriptionGiftEvent: %w", err)
	}
	fmt.Printf("Got channel.subscription.gift: %+v\n", ev)

	// If the subscriptions are tier 2 or 3, award 2x or 5x the points
	multiplier, err := getCreditMultiplierFromTier(ev.Tier)
	if err != nil {
		return nil
	}

	// Contact the auth server to request a short-lived JWT that will give us
	// authoritative access to the backend resources associated with the user who's
	// gifted the subs
	fmt.Printf("Requesting JWT in response to subs gift from user %s\n", ev.UserName)
	accessToken, err := h.authServiceClient.RequestServiceToken(ctx, auth.ServiceTokenRequest{
		Service: "showtime",
		User: auth.UserDetails{
			Id:          ev.UserID,
			Login:       ev.UserLogin,
			DisplayName: ev.UserName,
		},
	})
	if err != nil {
		return fmt.Errorf("RequestServiceToken failed: %w", err)
	}

	// Send a request to the ledger server to request that points be granted to the user
	fmt.Printf("Requesting credit of %d points x %d gift subs (before %.fx multiplier) to user %s\n", BasePointsForGiftSub, ev.Total, multiplier, ev.UserName)
	_, err = h.ledgerClient.RequestCreditFromGiftSub(ctx, accessToken, BasePointsForGiftSub, ev.Total, multiplier)
	if err != nil {
		return fmt.Errorf("RequestCreditFromGiftSub failed: %v", err)
	}

	// Finally, emit an alert so this user's new subscription will be acknowledged in
	// the on-stream overlay graphics
	h.alertsChan <- &alerts.Alert{
		Type: alerts.AlertTypeGiftSub,
		Data: alerts.AlertData{
			GiftSub: &alerts.AlertDataGiftSub{
				Username:         ev.UserName,
				NumSubscriptions: ev.Total,
			},
		},
	}

	return nil
}

// handleChannelSubscriptionMessageEvent responds to 'channel.subscription.message':
// "This subscription type sends a notification when a user sends a resubscription chat
// message in a specific channel."
//   - payload: {tier: "1000", message: {...}, cumulative_months: 15, streak_months?: 1,
//     duration_months: 6, ...}
func (h *Handler) handleChannelSubscriptionMessageEvent(ctx context.Context, data json.RawMessage) error {
	// Decode the JSON payload carried by the event
	var ev helix.EventSubChannelSubscriptionMessageEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelSubscriptionMessageEvent: %w", err)
	}
	fmt.Printf("Got channel.subscription.message: %+v\n", ev)

	// Record the fact that the user is now subscribed
	err := h.q.RecordViewerSubscribe(ctx, queries.RecordViewerSubscribeParams{
		TwitchUserID:      ev.UserID,
		TwitchDisplayName: ev.UserName,
	})
	if err != nil {
		return fmt.Errorf("RecordViewerSubscribe failed: %w\n", err)
	}

	// If the subscription is tier 2 or 3, award 2x or 5x the points
	multiplier, err := getCreditMultiplierFromTier(ev.Tier)
	if err != nil {
		return nil
	}

	// Contact the auth server to request a short-lived JWT that will give us
	// authoritative access to the backend resources associated with the user who's now
	// a subscriber
	fmt.Printf("Requesting JWT in response to resub message for user %s\n", ev.UserName)
	accessToken, err := h.authServiceClient.RequestServiceToken(ctx, auth.ServiceTokenRequest{
		Service: "showtime",
		User: auth.UserDetails{
			Id:          ev.UserID,
			Login:       ev.UserLogin,
			DisplayName: ev.UserName,
		},
	})
	if err != nil {
		return fmt.Errorf("RequestServiceToken failed: %w", err)
	}

	// Send a request to the ledger server to request that points be granted to the user
	fmt.Printf("Requesting credit of %d points (before %.fx multiplier) to user %s\n", BasePointsForSubscription, multiplier, ev.UserName)
	_, err = h.ledgerClient.RequestCreditFromSubscription(ctx, accessToken, BasePointsForSubscription, false, false, ev.Message.Text, multiplier)
	if err != nil {
		return fmt.Errorf("RequestCreditFromSubscription failed: %v", err)
	}

	// Finally, emit an alert so this user's resub message will be acknowledged in the
	// on-stream overlay graphics
	h.alertsChan <- &alerts.Alert{
		Type: alerts.AlertTypeSubscribe,
		Data: alerts.AlertData{
			Subscribe: &alerts.AlertDataSubscribe{
				Username:            ev.UserName,
				IsGift:              false,
				NumCumulativeMonths: ev.CumulativeMonths,
				Message:             ev.Message.Text,
			},
		},
	}

	return nil
}

func getCreditMultiplierFromTier(tier string) (float64, error) {
	// We expect the "tier" value in EventSub messages to be one of the following:
	switch tier {
	case "1000":
		// Tier 1 subs are the baseline at $5; fun points are credited 1x
		return 1, nil
	case "2000":
		// Tier 2 subs cost $10; fun point credits are doubled
		return 2, nil
	case "3000":
		// Tier 3 subs are $25; so subscribers get 5x fun points
		return 5, nil
	}

	// In the event of an unrecognized value, fail
	return 0, fmt.Errorf("unrecognized tier value '%s'", tier)
}
