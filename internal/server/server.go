package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/cors"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/eventsub"
)

type Server struct {
	http.Handler

	twitchAppClientId     string
	twitchAppClientSecret string
	extensionClientId     string
	webhookSecret         string
	q                     *queries.Queries
	eventsub              *eventsub.Client
	chat                  *chat.Client
	eventsChan            chan *chat.Event

	alerts     *subcriberChannels[*Alert]
	chatEvents *subcriberChannels[*chat.Event]
}

func New(ctx context.Context, twitchAppClientId string, twitchAppClientSecret string, extensionClientId string, webhookSecret string, q *queries.Queries, eventsubClient *eventsub.Client, chatClient *chat.Client, eventsChan chan *chat.Event) *Server {
	s := &Server{
		twitchAppClientId:     twitchAppClientId,
		twitchAppClientSecret: twitchAppClientSecret,
		extensionClientId:     extensionClientId,
		webhookSecret:         webhookSecret,
		q:                     q,
		eventsub:              eventsubClient,
		chat:                  chatClient,
		eventsChan:            eventsChan,
		alerts: &subcriberChannels[*Alert]{
			chs: make(map[int]chan *Alert),
		},
		chatEvents: &subcriberChannels[*chat.Event]{
			chs: make(map[int]chan *chat.Event),
		},
	}

	withCors := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://localhost:8080",
			fmt.Sprintf("https://%s.ext-twitch.tv", extensionClientId),
		},
		AllowedMethods: []string{http.MethodGet},
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleStatus)
	mux.HandleFunc("/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/auth/refresh", s.handleAuthRefresh)
	mux.HandleFunc("/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("/callback", s.handlePostCallback)
	mux.HandleFunc("/alerts", s.handleAlerts)
	mux.HandleFunc("/chat", s.handleChat)
	mux.HandleFunc("/view", s.handleView)
	s.Handler = withCors.Handler(mux)

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
