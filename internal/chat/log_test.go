package chat

import (
	"context"
	"testing"
	"time"

	irc "github.com/gempir/go-twitch-irc/v4"
	"github.com/stretchr/testify/assert"
)

func Test_Log(t *testing.T) {
	l := NewLog(16)
	assert.NotNil(t, l)

	ctx, cancel := context.WithCancel(context.Background())

	events := make([]*LogEvent, 0)
	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case event := <-l.events:
				events = append(events, event)
			}
		}
	}()

	l.handleMessage(irc.PrivateMessage{
		ID: "message-0",
		User: irc.User{
			ID:          "user-id-alice",
			DisplayName: "alice",
			Color:       "#ffcccc",
		},
		Message: "Hello, I am Alice",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-1",
		User: irc.User{
			ID:          "user-id-bob",
			DisplayName: "Bob",
			Color:       "#ccffcc",
		},
		Message: "Hello, I am Bob",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-2",
		User: irc.User{
			ID:          "user-id-alice",
			DisplayName: "alice",
			Color:       "#ffcccc",
		},
		Message: "Hello Bob, I am Alice",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-3",
		User: irc.User{
			ID:          "user-id-bob",
			DisplayName: "Bob",
			Color:       "#ccffcc",
		},
		Message: "yes, I know",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-4",
		User: irc.User{
			ID:          "user-id-charlie",
			DisplayName: "charlie",
			Color:       "#ccccff",
		},
		Message: "no need for snark; I am deleting your messages, Bob",
	})
	l.handleClearChatMessage(irc.ClearChatMessage{
		TargetUserID: "user-id-bob",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-5",
		User: irc.User{
			ID:          "user-id-dnitra",
			DisplayName: "Dnitra",
			Color:       "#ffffcc",
		},
		Message: "wow",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-6",
		User: irc.User{
			ID:          "user-id-dnitra",
			DisplayName: "Dnitra",
			Color:       "#ffffcc",
		},
		Message: "that was quite unnecessary, Charlie",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-7",
		User: irc.User{
			ID:          "user-id-charlie",
			DisplayName: "charlie",
			Color:       "#ccccff",
		},
		Message: "don't test me",
	})
	l.handleClearMessage(irc.ClearMessage{
		TargetMsgID: "message-6",
	})
	l.handleMessage(irc.PrivateMessage{
		ID: "message-8",
		User: irc.User{
			ID:          "user-id-bob",
			DisplayName: "Bob",
			Color:       "#ccffcc",
		},
		Message: "enough; just nuke everything",
	})
	l.handleClearChatMessage(irc.ClearChatMessage{})

	cancel()
	time.Sleep(10 * time.Millisecond)

	aliceSays := func(id string, text string) *LogEvent {
		return &LogEvent{
			Type: LogEventTypeMessage,
			Message: &LogMessage{
				ID:       id,
				Username: "alice",
				Color:    "#ffcccc",
				Text:     text,
				Emotes:   []EmoteDetails{},
			},
		}
	}
	bobSays := func(id string, text string) *LogEvent {
		return &LogEvent{
			Type: LogEventTypeMessage,
			Message: &LogMessage{
				ID:       id,
				Username: "Bob",
				Color:    "#ccffcc",
				Text:     text,
				Emotes:   []EmoteDetails{},
			},
		}
	}
	charlieSays := func(id string, text string) *LogEvent {
		return &LogEvent{
			Type: LogEventTypeMessage,
			Message: &LogMessage{
				ID:       id,
				Username: "charlie",
				Color:    "#ccccff",
				Text:     text,
				Emotes:   []EmoteDetails{},
			},
		}
	}
	dnitraSays := func(id string, text string) *LogEvent {
		return &LogEvent{
			Type: LogEventTypeMessage,
			Message: &LogMessage{
				ID:       id,
				Username: "Dnitra",
				Color:    "#ffffcc",
				Text:     text,
				Emotes:   []EmoteDetails{},
			},
		}
	}

	assert.Equal(t, []*LogEvent{
		aliceSays("message-0", "Hello, I am Alice"),
		bobSays("message-1", "Hello, I am Bob"),
		aliceSays("message-2", "Hello Bob, I am Alice"),
		bobSays("message-3", "yes, I know"),
		charlieSays("message-4", "no need for snark; I am deleting your messages, Bob"),
		{
			Type: LogEventTypeDeletion,
			Deletion: &LogDeletion{
				MessageIDs: []string{
					"message-1",
					"message-3",
				},
			},
		},
		dnitraSays("message-5", "wow"),
		dnitraSays("message-6", "that was quite unnecessary, Charlie"),
		charlieSays("message-7", "don't test me"),
		{
			Type: LogEventTypeDeletion,
			Deletion: &LogDeletion{
				MessageIDs: []string{
					"message-6",
				},
			},
		},
		bobSays("message-8", "enough; just nuke everything"),
		{
			Type: LogEventTypeClear,
		},
	}, events)
}
