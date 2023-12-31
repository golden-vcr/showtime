package chat

import (
	"context"
	"time"

	irc "github.com/gempir/go-twitch-irc/v4"
)

type Agent struct {
	client     *irc.Client
	connection *Connection
	log        *Log
}

func NewAgent(ctx context.Context, logBufferSize int, logEventsChan chan<- *LogEvent, channelName string, connectTimeout time.Duration) (*Agent, error) {
	log := NewLog(logBufferSize, logEventsChan)

	client := irc.NewAnonymousClient()
	client.OnPrivateMessage(log.handleMessage)
	client.OnClearMessage(log.handleClearMessage)
	client.OnClearChatMessage(log.handleClearChatMessage)
	client.Join(channelName)

	connection := NewConnection(client)
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	err := connection.Open(ctx)
	if err != nil {
		return nil, err
	}

	return &Agent{
		client:     client,
		connection: connection,
		log:        log,
	}, nil
}

func (a *Agent) GetStatus() error {
	return a.connection.GetStatus()
}

func (a *Agent) Disconnect() error {
	return a.connection.Close()
}
