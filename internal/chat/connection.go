package chat

import (
	"context"
	"errors"
	"fmt"
)

var ErrConnectionNotOpen = errors.New("not connected")

type IrcConnection interface {
	OnConnect(func())
	Connect() error
	Disconnect() error
}

type Connection struct {
	client IrcConnection

	connectErrChan chan error
	open           bool
	lastErr        error
}

func NewConnection(client IrcConnection) *Connection {
	return &Connection{
		client:         client,
		connectErrChan: make(chan error),
	}
}

func (c *Connection) GetStatus() error {
	if c.lastErr == nil && !c.open {
		return ErrConnectionNotOpen
	}
	return c.lastErr
}

func (c *Connection) Open(ctx context.Context) error {
	// Write nil (indicating no error) when connection succeeds, limited to the scope of
	// this function
	c.client.OnConnect(func() { c.connectErrChan <- nil })
	defer c.client.OnConnect(nil)

	// Connect() is blocking, so run it in a separate goroutine, and signal its return
	// value by writing to the error channel
	go func() {
		c.connectErrChan <- c.client.Connect()
	}()

	// If our context is canceled (e.g. because its timeout was exceeded before we
	// could connect), or if we got a connection error, return an error
	select {
	case <-ctx.Done():
		return fmt.Errorf("context canceled while waiting to connect: %v", ctx.Err())
	case err := <-c.connectErrChan:
		// If err is nil, our OnConnect callback was fired, and if non-nil, Connect
		// failed with an error
		if err != nil {
			// If we failed to connect, abort
			c.lastErr = err
			return err
		}
	}

	// If we got a nil result, we're now connected, and our client.Connect() goroutine
	// is still running: when it's done, it will write the result of the Connect call
	// into connectErrChan, and if that result is non-nil we want to store it as lastErr
	c.open = true
	go func() {
		select {
		case err := <-c.connectErrChan:
			if err != nil {
				c.open = false
				c.lastErr = err
			}
		}
	}()
	return nil
}

func (a *Connection) Close() error {
	if !a.open {
		return ErrConnectionNotOpen
	}
	a.open = false
	return a.client.Disconnect()
}
