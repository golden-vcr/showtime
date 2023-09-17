package server

import (
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

	alertChannels          map[int]chan *Alert
	alertChannelsMutex     sync.RWMutex
	nextAlertChannelHandle int
}

func New(eventsubClient *eventsub.Client, chatClient *chat.Client, webhookSecret string) *Server {
	s := &Server{
		eventsub:      eventsubClient,
		chat:          chatClient,
		webhookSecret: webhookSecret,
		alertChannels: make(map[int]chan *Alert),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleStatus)
	mux.HandleFunc("/callback", s.handlePostCallback)
	mux.HandleFunc("/alerts", s.handleAlerts)
	s.Handler = mux
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
