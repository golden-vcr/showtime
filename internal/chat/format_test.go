package chat

import (
	"testing"

	irc "github.com/gempir/go-twitch-irc/v4"
	"github.com/stretchr/testify/assert"
)

func Test_newMessageEvent(t *testing.T) {
	tests := []struct {
		name string
		m    *irc.PrivateMessage
		want *LogMessage
	}{
		{
			"basic message and user details are preserved",
			&irc.PrivateMessage{
				ID: "7070-22222222-1234",
				User: irc.User{
					ID:          "5550001",
					Name:        "bigjoe",
					DisplayName: "BigJoe",
					Color:       "#feefee",
				},
				Message: "hello world",
			},
			&LogMessage{
				ID:       "7070-22222222-1234",
				Username: "BigJoe",
				Color:    "#feefee",
				Text:     "hello world",
				Emotes:   []EmoteDetails{},
			},
		},
		{
			"whitespace is preserved",
			&irc.PrivateMessage{
				ID: "message-id",
				User: irc.User{
					DisplayName: "BigJoe",
					Color:       "#feefee",
				},
				Message: " hello...    world ! ",
			},
			&LogMessage{
				ID:       "message-id",
				Username: "BigJoe",
				Color:    "#feefee",
				Text:     " hello...    world ! ",
				Emotes:   []EmoteDetails{},
			},
		},
		{
			"dollar signs are escaped",
			&irc.PrivateMessage{
				ID: "message-id",
				User: irc.User{
					DisplayName: "BigJoe",
					Color:       "#feefee",
				},
				Message: "Lincoln is on the $5 bill",
			},
			&LogMessage{
				ID:       "message-id",
				Username: "BigJoe",
				Color:    "#feefee",
				Text:     "Lincoln is on the $$5 bill",
				Emotes:   []EmoteDetails{},
			},
		},
		{
			"emotes are replaced with $0, $1, etc.",
			&irc.PrivateMessage{
				ID: "message-id",
				User: irc.User{
					DisplayName: "BigJoe",
					Color:       "#feefee",
				},
				Message: "Lincoln presidAbe presidAbe is on the $5 bill FrankerZ !",
				Emotes: []*irc.Emote{
					{
						Name: "presidAbe",
						ID:   "emote-of-lincoln",
					},
					{
						Name: "FrankerZ",
						ID:   "emote-of-dog",
					},
				},
			},
			&LogMessage{
				ID:       "message-id",
				Username: "BigJoe",
				Color:    "#feefee",
				Text:     "Lincoln $0 $0 is on the $$5 bill $1 !",
				Emotes: []EmoteDetails{
					{
						Name: "presidAbe",
						Url:  "https://static-cdn.jtvnw.net/emoticons/v2/emote-of-lincoln/default/dark/1.0",
					},
					{
						Name: "FrankerZ",
						Url:  "https://static-cdn.jtvnw.net/emoticons/v2/emote-of-dog/default/dark/1.0",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newMessageEvent(tt.m)
			assert.Equal(t, LogEventTypeMessage, got.Type)
			assert.Equal(t, tt.want, got.Message)
		})
	}
}
