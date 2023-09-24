package server

import (
	"fmt"
	"strings"

	irc "github.com/gempir/go-twitch-irc/v4"
)

type ChatLine struct {
	Name   string         `json:"name"`
	Color  string         `json:"color"`
	Text   string         `json:"text"`
	Emotes []EmoteDetails `json:"emotes"`
}

type EmoteDetails struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

func (s *Server) handleMessage(message *irc.PrivateMessage) {
	text, emoteNames := substituteEmotes(message)
	emotes := make([]EmoteDetails, 0, len(emoteNames))
	for _, emoteName := range emoteNames {
		emotes = append(emotes, EmoteDetails{
			Name: emoteName,
			Url:  "",
		})
	}
	s.broadcastChatLine(&ChatLine{
		Name:   message.User.DisplayName,
		Color:  message.User.Color,
		Text:   text,
		Emotes: emotes,
	})
}

func substituteEmotes(message *irc.PrivateMessage) (string, []string) {
	tokens := strings.Split(strings.ReplaceAll(message.Message, "$", "$$"), " ")
	emoteNames := make([]string, 0, len(message.Emotes))
	for emoteIndex, emote := range message.Emotes {
		for tokenIndex := 0; tokenIndex < len(tokens); tokenIndex++ {
			if tokens[tokenIndex] == emote.Name {
				tokens[tokenIndex] = fmt.Sprintf("$%d", emoteIndex)
			}
		}
		emoteNames = append(emoteNames, emote.Name)
	}
	return strings.Join(tokens, " "), emoteNames
}
