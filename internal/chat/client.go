package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	irc "github.com/gempir/go-twitch-irc/v4"
	"golang.org/x/sync/errgroup"
)

var ErrNotYetConnected = errors.New("not yet connected")

type Client struct {
	channelName  string
	messagesChan chan irc.PrivateMessage

	client         *irc.Client
	lastConnectErr error
}

func NewClient(ctx context.Context, channelName string, messagesChan chan irc.PrivateMessage) (*Client, error) {
	c := &Client{
		channelName:    channelName,
		messagesChan:   messagesChan,
		client:         irc.NewAnonymousClient(),
		lastConnectErr: ErrNotYetConnected,
	}
	c.client.OnConnect(func() {
		fmt.Printf("Connected to Twitch IRC channel for '%s'\n", c.channelName)
		c.lastConnectErr = nil
	})
	c.client.OnPrivateMessage(c.handleMessage)
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

func (c *Client) connect(ctx context.Context) {
	var wg errgroup.Group
	wg.Go(c.client.Connect)
	wg.Go(func() error {
		select {
		case <-ctx.Done():
			return c.client.Disconnect()
		}
	})
	go func() {
		c.lastConnectErr = wg.Wait()
		fmt.Printf("IRC client Connect() returned: %v\n", c.lastConnectErr)
	}()
}

func (c *Client) GetStatus() error {
	return c.lastConnectErr
}

func (c *Client) handleMessage(message irc.PrivateMessage) {
	fmt.Printf("CHAT | %s: %s\n", message.User.Name, message.Message)
	for _, emote := range message.Emotes {
		fmt.Printf("   e | %v\n", emote)
	}
	for k, v := range message.Tags {
		fmt.Printf("   t | %s: %s\n", k, v)
	}
	c.messagesChan <- message
}
