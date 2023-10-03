package chat

import (
	"fmt"

	irc "github.com/gempir/go-twitch-irc/v4"
)

type Log struct {
	events chan *LogEvent
	buffer *messageBuffer
}

func NewLog(numMessagesToBuffer int) *Log {
	return &Log{
		events: make(chan *LogEvent, 32),
		buffer: newMessageBuffer(numMessagesToBuffer),
	}
}

// handleMessage is called in response to an IRC PRIVMSG
func (a *Log) handleMessage(m irc.PrivateMessage) {
	fmt.Printf("CHAT | (m:%s u:%s) | %s: %s\n", m.ID, m.User.ID, m.User.Name, m.Message)
	if event := newMessageEvent(&m); event != nil {
		a.buffer.add(m.User.ID, m.ID)
		a.events <- event
	}
}

// handleClearMessage is called in response to an IRC CLEARMSG, which targets a single
// message ID for deletion
func (a *Log) handleClearMessage(m irc.ClearMessage) {
	fmt.Printf("CLEAR | (m:%s)\n", m.TargetMsgID)
	event := &LogEvent{
		Type: LogEventTypeDeletion,
		Deletion: &LogDeletion{
			MessageIDs: []string{m.TargetMsgID},
		},
	}
	a.events <- event
}

// handleClearChatMessage is called in response to an IRC CLEARCHAT, which either
// clears the entire chat log (if no target user ID is specified) or targets all
// messages sent by the target user for deletion
func (a *Log) handleClearChatMessage(m irc.ClearChatMessage) {
	if m.TargetUserID != "" {
		fmt.Printf("CLEAR | (u:%s)\n", m.TargetUserID)
		messageIds := a.buffer.resolveMessageIds(m.TargetUserID)
		if len(messageIds) > 0 {
			event := &LogEvent{
				Type: LogEventTypeDeletion,
				Deletion: &LogDeletion{
					MessageIDs: messageIds,
				},
			}
			a.events <- event
		}
	} else {
		fmt.Printf("CLEAR ALL\n")
		a.events <- &LogEvent{Type: LogEventTypeClear}
	}
}
