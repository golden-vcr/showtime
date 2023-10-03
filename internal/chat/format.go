package chat

import (
	"fmt"
	"strings"

	irc "github.com/gempir/go-twitch-irc/v4"
)

// newMessageEvent constructs an LogEvent with type 'message' given an IRC PRIVMSG, or
// nil if the IRC message should not result in a new chat line being displayed
func newMessageEvent(m *irc.PrivateMessage) *LogEvent {
	text, emotes := substituteEmotes(m)
	return &LogEvent{
		Type: LogEventTypeMessage,
		Message: &LogMessage{
			ID:       m.ID,
			Username: m.User.DisplayName,
			Color:    m.User.Color,
			Text:     text,
			Emotes:   emotes,
		},
	}
}

func substituteEmotes(m *irc.PrivateMessage) (string, []EmoteDetails) {
	tokens := strings.Split(strings.ReplaceAll(m.Message, "$", "$$"), " ")
	emotes := make([]EmoteDetails, 0, len(m.Emotes))
	for emoteIndex, emote := range m.Emotes {
		for tokenIndex := 0; tokenIndex < len(tokens); tokenIndex++ {
			if tokens[tokenIndex] == emote.Name {
				tokens[tokenIndex] = fmt.Sprintf("$%d", emoteIndex)
			}
		}
		emotes = append(emotes, EmoteDetails{
			Name: emote.Name,
			Url:  fmt.Sprintf("https://static-cdn.jtvnw.net/emoticons/v2/%s/default/dark/1.0", emote.ID),
		})
	}
	return strings.Join(tokens, " "), emotes
}
