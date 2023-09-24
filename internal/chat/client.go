package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	irc "github.com/gempir/go-twitch-irc/v4"
	"golang.org/x/sync/errgroup"
)

// ErrNotYetConnected is the initial error state for the client, until it connects or
// encounters another error
var ErrNotYetConnected = errors.New("not yet connected")

// NumMessagesToBuffer controls how many recent messages we buffer in-memory; this
// should be >= the maximum number of messages that the chat log UI keeps onscreen
const NumMessagesToBuffer = 64

// Client wraps an IRC chat client, providing us with a stream of chat.Event structs
// which we can fan out to all interested HTTP clients
type Client struct {
	// channelName identifies the Twitch channel we should connect to
	channelName string

	// eventsChan will receive an Event in repsonse to actions taken in Twitch chat
	eventsChan chan *Event

	// client is the go-twitch-irc client that gives us direct access to IRC chat
	// messages (e.g. PRIVMSG, CLEARCHAT, CLEARMSG, etc.), enriched with
	// Twitch-specific metadata
	client *irc.Client

	// lastConnectErr surfaces the connection state: if ErrNotYetConnected, we're not
	// yet initialized; if nil, we're connected; if any other error; we failed to
	// connect or we've lost our connection
	lastConnectErr error

	// buffer keeps track of user IDs associated with the N most recent messages (where
	// N is NumMessagesToBuffer), so that when we need to clear all messages associated
	// with a specific user, we can resolve a set of message IDs from their user ID
	buffer *messageBuffer
}

func NewClient(ctx context.Context, channelName string, eventsChan chan *Event) (*Client, error) {
	c := &Client{
		channelName:    channelName,
		eventsChan:     eventsChan,
		client:         irc.NewAnonymousClient(),
		lastConnectErr: ErrNotYetConnected,
		buffer:         newMessageBuffer(NumMessagesToBuffer),
	}
	c.client.OnConnect(func() {
		fmt.Printf("Connected to Twitch IRC channel for '%s'\n", c.channelName)
		c.lastConnectErr = nil
	})
	c.client.OnPrivateMessage(c.handleMessage)
	c.client.OnClearMessage(c.handleClearMessage)
	c.client.OnClearChatMessage(c.handleClearChatMessage)
	c.client.Join(c.channelName)
	c.connect(ctx)

	establishConnectionCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	for {
		select {
		case <-establishConnectionCtx.Done():
			if c.lastConnectErr != nil {
				return nil, fmt.Errorf("Failed to establish initial connection to Twitch chat: %w", c.lastConnectErr)
			} else {
				return nil, fmt.Errorf("Failed to establish initial connection to Twitch chat")
			}
		case <-time.After(10 * time.Millisecond):
			if c.lastConnectErr == nil {
				return c, nil
			}
		}
	}
}

// connect kicks off a goroutine that will run the IRC client indefinitely,
// disconnecting when the provided context is canceled, and storing any result received
// from client.Connect() as lastConnectErr
func (c *Client) connect(ctx context.Context) {
	// Use an errgroup, which will halt as soon as any child goroutine returns an error
	var wg errgroup.Group

	// client.Connect() is blocking; run it in separate goroutine
	wg.Go(c.client.Connect)

	// client does not support contexts; run another goroutine to automatically
	// disconnect when the context is canceled
	wg.Go(func() error {
		select {
		case <-ctx.Done():
			return c.client.Disconnect()
		}
	})

	// Outside of the errgroup, run another goroutine to await the result returned by
	// either client.Connect() or client.Disconnect()
	go func() {
		c.lastConnectErr = wg.Wait()
		fmt.Printf("IRC client Connect() returned: %v\n", c.lastConnectErr)
	}()
}

// GetStatus returns the current status of the IRC client: nil if connected without
// error; non-nil if unable to connect
func (c *Client) GetStatus() error {
	return c.lastConnectErr
}

// handleMessage is called in response to an IRC PRIVMSG
func (c *Client) handleMessage(m irc.PrivateMessage) {
	fmt.Printf("CHAT | (m:%s u:%s) | %s: %s\n", m.ID, m.User.ID, m.User.Name, m.Message)
	if event := newMessageEvent(&m); event != nil {
		c.buffer.add(m.User.ID, m.ID)
		c.eventsChan <- event
	}
}

// handleClearMessage is called in response to an IRC CLEARMSG, which targets a single
// message ID for deletion
func (c *Client) handleClearMessage(m irc.ClearMessage) {
	fmt.Printf("CLEAR | (m:%s)\n", m.TargetMsgID)
	event := &Event{
		Type: EventTypeDeletion,
		Deletion: &Deletion{
			MessageIDs: []string{m.TargetMsgID},
		},
	}
	c.eventsChan <- event
}

// handleClearChatMessage is called in response to an IRC CLEARCHAT, which either
// clears the entire chat log (if no target user ID is specified) or targets all
// messages sent by the target user for deletion
func (c *Client) handleClearChatMessage(m irc.ClearChatMessage) {
	if m.TargetUserID != "" {
		fmt.Printf("CLEAR | (u:%s)\n", m.TargetUserID)
		messageIds := c.buffer.resolveMessageIds(m.TargetUserID)
		if len(messageIds) > 0 {
			event := &Event{
				Type: EventTypeDeletion,
				Deletion: &Deletion{
					MessageIDs: messageIds,
				},
			}
			c.eventsChan <- event
		}

	} else {
		fmt.Printf("CLEAR ALL\n")
		c.eventsChan <- &Event{Type: EventTypeClear}
	}
}
