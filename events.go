package showtime

import (
	"sort"

	"github.com/golden-vcr/showtime/internal/eventsub"
	"github.com/nicklaw5/helix/v2"
)

// RequiredSubscriptions declares all of the subscriptions that our showtime API
// supports
var RequiredSubscriptions = []eventsub.Subscription{
	{
		Type:    helix.EventSubTypeChannelUpdate,
		Version: "2",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: "{{.ChannelUserId}}",
		},
	},
	{
		Type:    helix.EventSubTypeChannelFollow,
		Version: "2",
		Condition: helix.EventSubCondition{
			BroadcasterUserID: "{{.ChannelUserId}}",
			ModeratorUserID:   "{{.ChannelUserId}}",
		},
		RequiredScopes: []string{
			"moderator:read:followers",
		},
	},
}

// GetRequiredUserScopes returns a list of all OAuth scopes that the connected Twitch
// user (i.e. the broadcaster) must authorize for our app: we'll only be able to create
// subscriptions via the EventSub API (using our app access token) once user access has
// been granted
func GetRequiredUserScopes() []string {
	scopes := make(map[string]struct{})
	for i := range RequiredSubscriptions {
		for _, scope := range RequiredSubscriptions[i].RequiredScopes {
			scopes[scope] = struct{}{}
		}
	}
	scopesArray := make([]string, 0, len(scopes))
	for k := range scopes {
		scopesArray = append(scopesArray, k)
	}
	sort.Strings(scopesArray)
	return scopesArray
}