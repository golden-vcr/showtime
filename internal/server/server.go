package server

import (
	"context"
	"net/http"
	"sync"

	irc "github.com/gempir/go-twitch-irc/v4"

	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/eventsub"
)

type Server struct {
	http.Handler

	eventsub      *eventsub.Client
	chat          *chat.Client
	webhookSecret string
	messagesChan  chan irc.PrivateMessage

	alertChannels          map[int]chan *Alert
	alertChannelsMutex     sync.RWMutex
	nextAlertChannelHandle int

	chatChannels          map[int]chan *ChatLine
	chatChannelsMutex     sync.RWMutex
	nextChatChannelHandle int
}

func New(ctx context.Context, eventsubClient *eventsub.Client, chatClient *chat.Client, webhookSecret string, messagesChan chan irc.PrivateMessage) *Server {
	s := &Server{
		eventsub:      eventsubClient,
		chat:          chatClient,
		webhookSecret: webhookSecret,
		messagesChan:  messagesChan,
		alertChannels: make(map[int]chan *Alert),
		chatChannels:  make(map[int]chan *ChatLine),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleStatus)
	mux.HandleFunc("/callback", s.handlePostCallback)
	mux.HandleFunc("/alerts", s.handleAlerts)
	mux.HandleFunc("/chat", s.handleChat)
	s.Handler = mux

	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case message := <-s.messagesChan:
				s.handleMessage(&message)
			}
		}
	}()

	return s
}

func (s *Server) subscribeToAlerts(ch chan *Alert) int {
	s.alertChannelsMutex.Lock()
	defer s.alertChannelsMutex.Unlock()

	handle := s.nextAlertChannelHandle
	s.nextAlertChannelHandle++
	s.alertChannels[handle] = ch
	return handle
}

func (s *Server) unsubscribeFromAlerts(handle int) {
	s.alertChannelsMutex.Lock()
	defer s.alertChannelsMutex.Unlock()

	delete(s.alertChannels, handle)
}

func (s *Server) broadcastAlert(alert *Alert) {
	s.alertChannelsMutex.RLock()
	defer s.alertChannelsMutex.RUnlock()

	for _, ch := range s.alertChannels {
		ch <- alert
	}
}

func (s *Server) subscribeToChat(ch chan *ChatLine) int {
	s.chatChannelsMutex.Lock()
	defer s.chatChannelsMutex.Unlock()

	handle := s.nextChatChannelHandle
	s.nextChatChannelHandle++
	s.chatChannels[handle] = ch
	return handle
}

func (s *Server) unsubscribeFromChat(handle int) {
	s.chatChannelsMutex.Lock()
	defer s.chatChannelsMutex.Unlock()

	delete(s.chatChannels, handle)
}

func (s *Server) broadcastChatLine(line *ChatLine) {
	s.chatChannelsMutex.RLock()
	defer s.chatChannelsMutex.RUnlock()

	for _, ch := range s.chatChannels {
		ch <- line
	}
}
