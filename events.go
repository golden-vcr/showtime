package showtime

import (
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/nicklaw5/helix/v2"
)

// RequiredSubscriptions declares all of the subscriptions that our showtime API
// supports
var RequiredSubscriptions = []events.RequiredSubscription{
	{
		Type:    helix.EventSubTypeChannelUpdate,
		Version: "2",
		TemplatedCondition: helix.EventSubCondition{
			BroadcasterUserID: "{{.ChannelUserId}}",
		},
	},
	{
		Type:    helix.EventSubTypeStreamOnline,
		Version: "1",
		TemplatedCondition: helix.EventSubCondition{
			BroadcasterUserID: "{{.ChannelUserId}}",
		},
	},
	{
		Type:    helix.EventSubTypeStreamOffline,
		Version: "1",
		TemplatedCondition: helix.EventSubCondition{
			BroadcasterUserID: "{{.ChannelUserId}}",
		},
	},
	{
		Type:    helix.EventSubTypeChannelFollow,
		Version: "2",
		TemplatedCondition: helix.EventSubCondition{
			BroadcasterUserID: "{{.ChannelUserId}}",
			ModeratorUserID:   "{{.ChannelUserId}}",
		},
		RequiredScopes: []string{
			"moderator:read:followers",
		},
	},
	{
		Type:    helix.EventSubTypeChannelRaid,
		Version: "1",
		TemplatedCondition: helix.EventSubCondition{
			ToBroadcasterUserID: "{{.ChannelUserId}}",
		},
	},
}
