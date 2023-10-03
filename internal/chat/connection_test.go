package chat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Connection(t *testing.T) {
	t.Run("normal open/close works and updates status", func(t *testing.T) {
		conn := NewConnection(&connectionTestClient{})
		assert.False(t, conn.open)
		assert.ErrorIs(t, conn.GetStatus(), ErrConnectionNotOpen)

		err := conn.Open(context.Background())
		assert.NoError(t, err)
		assert.True(t, conn.open)
		assert.NoError(t, conn.GetStatus())

		err = conn.Close()
		assert.NoError(t, err)
		assert.False(t, conn.open)
		assert.ErrorIs(t, conn.GetStatus(), ErrConnectionNotOpen)
	})
	t.Run("attempting to close when not open returns ErrConnectionNotOpen", func(t *testing.T) {
		conn := NewConnection(&connectionTestClient{})
		err := conn.Close()
		assert.ErrorIs(t, err, ErrConnectionNotOpen)
	})
	t.Run("error on connect is signaled via Open", func(t *testing.T) {
		conn := NewConnection(&connectionTestClient{
			connectEarlyError: fmt.Errorf("mock error"),
		})
		err := conn.Open(context.Background())
		assert.ErrorContains(t, err, "mock error")
	})
	t.Run("error on disconnect is signaled via Close", func(t *testing.T) {
		conn := NewConnection(&connectionTestClient{
			disconnectErr: fmt.Errorf("mock error"),
		})
		err := conn.Open(context.Background())
		assert.NoError(t, err)
		err = conn.Close()
		assert.ErrorContains(t, err, "mock error")
	})
	t.Run("Open aborts when context is canceled during connection", func(t *testing.T) {
		conn := NewConnection(&connectionTestClient{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := conn.Open(ctx)
		assert.ErrorContains(t, err, "context canceled")
	})
	t.Run("connection is marked closed and error is recorded when client.Connect returns", func(t *testing.T) {
		conn := NewConnection(&connectionTestClient{
			connectReturnDelay: 5 * time.Millisecond,
			connectReturnErr:   fmt.Errorf("mock disconnect"),
		})
		err := conn.Open(context.Background())
		assert.NoError(t, err)
		assert.True(t, conn.open)
		assert.NoError(t, conn.GetStatus())

		var status error
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
			case <-time.After(5 * time.Millisecond):
				if !conn.open {
					status = conn.GetStatus()
					done = true
				}
			}
		}
		assert.NotNil(t, status)
		assert.ErrorContains(t, status, "mock disconnect")
	})
}

type connectionTestClient struct {
	connectEarlyError error
	disconnectErr     error

	connectReturnDelay time.Duration
	connectReturnErr   error

	onConnectCallback func()
}

func (c *connectionTestClient) Connect() error {
	if c.connectEarlyError != nil {
		return c.connectEarlyError
	}
	if c.onConnectCallback != nil {
		c.onConnectCallback()
	}

	if c.connectReturnDelay > 0 {
		time.Sleep(c.connectReturnDelay)
	} else {
		time.Sleep(10 * time.Millisecond)
	}
	return c.connectReturnErr
}

func (c *connectionTestClient) Disconnect() error {
	if c.disconnectErr != nil {
		return c.disconnectErr
	}
	return nil
}

func (c *connectionTestClient) OnConnect(callback func()) {
	c.onConnectCallback = callback
}

var _ IrcConnection = (*connectionTestClient)(nil)
