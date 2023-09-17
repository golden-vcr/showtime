package server

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/nicklaw5/helix/v2"
)

var ErrUnsupportedEventType = errors.New("unsupported event type")

type Event struct {
	Type          string
	ChannelUpdate *ChannelUpdateEvent
	ChannelFollow *ChannelFollowEvent
	ChannelRaid   *ChannelRaidEvent
}

type ChannelUpdateEvent struct {
	BroadcasterUserId           string   `json:"broadcaster_user_id"`
	BroadcasterUserLogin        string   `json:"broadcaster_user_login"`
	BroadcasterUserName         string   `json:"broadcaster_user_name"`
	Title                       string   `json:"title"`
	Language                    string   `json:"language"`
	CategoryId                  string   `json:"category_id"`
	CategoryName                string   `json:"category_name"`
	ContentClassificationLabels []string `json:"content_classification_labels"`
}

type ChannelFollowEvent struct {
	UserId               string    `json:"user_id"`
	UserLogin            string    `json:"user_login"`
	UserName             string    `json:"user_name"`
	BroadcasterUserId    string    `json:"broadcaster_user_id"`
	BroadcasterUserLogin string    `json:"broadcaster_user_login"`
	BroadcasterUserName  string    `json:"broadcaster_user_name"`
	FollowedAt           time.Time `json:"followed_at"`
}

type ChannelRaidEvent struct {
	UserId               string `json:"user_id"`
	UserLogin            string `json:"user_login"`
	UserName             string `json:"user_name"`
	BroadcasterUserId    string `json:"broadcaster_user_id"`
	BroadcasterUserLogin string `json:"broadcaster_user_login"`
	BroadcasterUserName  string `json:"broadcaster_user_name"`
	Viewers              int    `json:"viewers"`
}

func parseEvent(subscription *helix.EventSubSubscription, data json.RawMessage) (*Event, error) {
	event := &Event{Type: subscription.Type}
	switch event.Type {
	case helix.EventSubTypeChannelUpdate:
		event.ChannelUpdate = &ChannelUpdateEvent{}
		if err := json.Unmarshal(data, &event.ChannelUpdate); err != nil {
			return nil, err
		}
		return event, nil
	case helix.EventSubTypeChannelFollow:
		event.ChannelFollow = &ChannelFollowEvent{}
		if err := json.Unmarshal(data, &event.ChannelFollow); err != nil {
			return nil, err
		}
		return event, nil
	case helix.EventSubTypeChannelRaid:
		event.ChannelRaid = &ChannelRaidEvent{}
		if err := json.Unmarshal(data, &event.ChannelRaid); err != nil {
			return nil, err
		}
		return event, nil
	}
	return nil, ErrUnsupportedEventType
}
