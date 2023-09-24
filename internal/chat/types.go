package chat

import (
	"fmt"
	"strings"

	irc "github.com/gempir/go-twitch-irc/v4"
)

// EventType is an abstraction on top of IRC messages, presenting the frontend with a
// simplified set of chat events that are germane to rendering the chat log
type EventType string

const (
	// EventTypeMessage indicates that a new chat line should be displayed
	EventTypeMessage EventType = "message"
	// EventTypeDeletion indicates that one or more previous lines should be deleted
	EventTypeDeletion EventType = "deletion"
	// EventTypeClear indicates that all lines should be deleted from the log
	EventTypeClear EventType = "clear"
)

// Event is something occurring in Twitch chat that the frontend needs to know about
type Event struct {
	Type     EventType `json:"type"`
	Message  *Message  `json:"message,omitempty"`
	Deletion *Deletion `json:"deletion,omitempty"`
}

// Message is the payload for an event with type 'message'
type Message struct {
	ID       string         `json:"id"`
	Username string         `json:"username"`
	Color    string         `json:"color"`
	Text     string         `json:"text"`
	Emotes   []EmoteDetails `json:"emotes"`
}

type EmoteDetails struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// Deletion is the payload for an event with type 'deletion'
type Deletion struct {
	MessageIDs []string `json:"messageIds"`
}

// newMessageEvent constructs an Event with type 'message' given an IRC PRIVMSG, or nil
// if the IRC message should not result in a new chat line being displayed
func newMessageEvent(m *irc.PrivateMessage) *Event {
	text, emoteNames := substituteEmotes(m)
	emotes := make([]EmoteDetails, 0, len(emoteNames))
	for _, emoteName := range emoteNames {
		emotes = append(emotes, EmoteDetails{
			Name: emoteName,
			Url:  "",
		})
	}
	return &Event{
		Type: EventTypeMessage,
		Message: &Message{
			ID:       m.ID,
			Username: m.User.DisplayName,
			Color:    m.User.Color,
			Text:     text,
			Emotes:   emotes,
		},
	}
}

func substituteEmotes(m *irc.PrivateMessage) (string, []string) {
	tokens := strings.Split(strings.ReplaceAll(m.Message, "$", "$$"), " ")
	emoteNames := make([]string, 0, len(m.Emotes))
	for emoteIndex, emote := range m.Emotes {
		for tokenIndex := 0; tokenIndex < len(tokens); tokenIndex++ {
			if tokens[tokenIndex] == emote.Name {
				tokens[tokenIndex] = fmt.Sprintf("$%d", emoteIndex)
			}
		}
		emoteNames = append(emoteNames, emote.Name)
	}
	return strings.Join(tokens, " "), emoteNames
}
