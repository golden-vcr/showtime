package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/eventsub"
)

type Server struct {
	http.Handler

	eventsub      *eventsub.Client
	chat          *chat.Client
	webhookSecret string
	eventsChan    chan *chat.Event

	alertChannels          map[int]chan *Alert
	alertChannelsMutex     sync.RWMutex
	nextAlertChannelHandle int

	chatChannels          map[int]chan *chat.Event
	chatChannelsMutex     sync.RWMutex
	nextChatChannelHandle int
}

func New(ctx context.Context, eventsubClient *eventsub.Client, chatClient *chat.Client, webhookSecret string, eventsChan chan *chat.Event) *Server {
	s := &Server{
		eventsub:      eventsubClient,
		chat:          chatClient,
		webhookSecret: webhookSecret,
		eventsChan:    eventsChan,
		alertChannels: make(map[int]chan *Alert),
		chatChannels:  make(map[int]chan *chat.Event),
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
			case event := <-s.eventsChan:
				s.broadcastChatEvent(event)
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

func (s *Server) subscribeToChat(ch chan *chat.Event) int {
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

func (s *Server) broadcastChatEvent(event *chat.Event) {
	s.chatChannelsMutex.RLock()
	defer s.chatChannelsMutex.RUnlock()

	for _, ch := range s.chatChannels {
		ch <- event
	}
}
