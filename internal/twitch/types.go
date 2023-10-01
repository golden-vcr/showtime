package twitch

import "github.com/nicklaw5/helix/v2"

// SubscriptionReader represents the subset of Twitch Helix API operations required to
// view the state of EventSub subscriptions, with read-only access
type SubscriptionReader interface {
	GetEventSubSubscriptions(params *helix.EventSubSubscriptionsParams) (*helix.EventSubSubscriptionsResponse, error)
}
