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

	alerts     *subcriberChannels[*Alert]
	chatEvents *subcriberChannels[*chat.Event]
}

func New(ctx context.Context, eventsubClient *eventsub.Client, chatClient *chat.Client, webhookSecret string, eventsChan chan *chat.Event) *Server {
	s := &Server{
		eventsub:      eventsubClient,
		chat:          chatClient,
		webhookSecret: webhookSecret,
		eventsChan:    eventsChan,
		alerts: &subcriberChannels[*Alert]{
			chs: make(map[int]chan *Alert),
		},
		chatEvents: &subcriberChannels[*chat.Event]{
			chs: make(map[int]chan *chat.Event),
		},
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
				s.chatEvents.broadcast(event)
			}
		}
	}()

	return s
}

type subcriberChannels[T any] struct {
	chs        map[int]chan T
	mu         sync.RWMutex
	nextHandle int
}

func (s *subcriberChannels[T]) register(ch chan T) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	handle := s.nextHandle
	s.chs[handle] = ch
	s.nextHandle++
	return handle
}

func (s *subcriberChannels[T]) unregister(handle int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.chs, handle)
}

func (s *subcriberChannels[T]) broadcast(message T) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.chs {
		ch <- message
	}
}
