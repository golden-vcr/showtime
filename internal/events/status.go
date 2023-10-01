package events

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golden-vcr/showtime/internal/twitch"
	"github.com/nicklaw5/helix/v2"
)

var ErrFailedToGetSubscriptions = errors.New("Unable to retrieve subscription details from the Twitch API. This may be due to a disruption in service from Twitch itself, or there may be a problem with the Golden VCR server.")
var ErrNoSubscriptionsExist = errors.New("No Twitch event subscriptions are enabled. The Golden VCR server may not yet be fully connected to the Golden VCR Twitch account.")
var ErrFailedToReconcileSubscriptions = errors.New("Unable to ascertain status of required Twitch event subscriptions. This may indicate a problem with the Golden VCR server.")
var ErrMissingSubscriptions = errors.New("One or more required Twitch event subscriptions do not yet exist. The Golden VCR server may not be receiving all required data from Twitch.")
var ErrSubscriptionsDisabled = errors.New("One or more required Twitch event subscriptions are disabled. The Golden VCR server may not be receiving all required data from Twitch.")

func VerifySubscriptionStatus(c twitch.SubscriptionReader, required []RequiredSubscription, channelUserId string, webhookCallbackUrl string) (error, error) {
	// Get a list of all relevant subscriptions for our channel and callback
	owned, err := GetOwnedSubscriptions(c, channelUserId, webhookCallbackUrl)
	if err != nil {
		return ErrFailedToGetSubscriptions, err
	}

	// We must have at least one subscriptions
	if len(owned) == 0 {
		return ErrNoSubscriptionsExist, nil
	}

	// We must have an existing subscription for all required subscriptions
	reconciled, err := ReconcileRequiredSubscriptions(required, owned, channelUserId)
	if err != nil {
		return ErrFailedToReconcileSubscriptions, err
	}
	if len(reconciled.ToCreate) > 0 {
		return ErrMissingSubscriptions, nil
	}

	// All relevant subscriptions must have the 'enabled' status
	for _, existing := range reconciled.Existing {
		if existing.Value.Status != helix.EventSubStatusEnabled {
			return ErrSubscriptionsDisabled, nil
		}
	}

	// All required Twitch EventSub subscriptions are healthy
	return nil, nil
}

// GetOwnedSubscriptions queries the Twitch API to find all relevant EventSub
// subscriptions that are registered with the given UserID and webhook callback URL
func GetOwnedSubscriptions(c twitch.SubscriptionReader, channelUserId string, webhookCallbackUrl string) ([]helix.EventSubSubscription, error) {
	subscriptions := make([]helix.EventSubSubscription, 0)
	params := &helix.EventSubSubscriptionsParams{
		UserID: channelUserId,
	}
	for {
		// Query the Twitch API for a list of our EventSub subscriptions
		r, err := c.GetEventSubSubscriptions(params)
		if err != nil {
			return nil, err
		}
		if r.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("got response %d from get subscriptions request: %s", r.StatusCode, r.ErrorMessage)
		}

		for i := range r.Data.EventSubSubscriptions {
			// Ignore any subscriptions that don't hit our webhook API
			subscription := r.Data.EventSubSubscriptions[i]
			if subscription.Transport.Method != "webhook" {
				continue
			}
			if subscription.Transport.Callback != webhookCallbackUrl {
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

func ReconcileRequiredSubscriptions(required []RequiredSubscription, owned []helix.EventSubSubscription, channelUserId string) (*ReconcileResult, error) {
	params := RequiredSubscriptionConditionParams{
		ChannelUserId: channelUserId,
	}

	requiredSubscriptionsByExistingId := make(map[string]RequiredSubscription)
	requiredSubscriptionsThatDoNotExist := make([]RequiredSubscription, 0)
	for i, requiredSubscription := range required {
		requiredCondition, err := params.Format(&requiredSubscription.TemplatedCondition)
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

type ReconcileResult struct {
	ToDelete []helix.EventSubSubscription
	ToCreate []RequiredSubscription
	Existing []ExistingSubscription
}

type ExistingSubscription struct {
	Value    helix.EventSubSubscription
	Required RequiredSubscription
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
