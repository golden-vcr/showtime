package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/nicklaw5/helix/v2"
	"github.com/rs/cors"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/golden-vcr/showtime/internal/auth"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/golden-vcr/showtime/internal/twitch"
)

type Server struct {
	http.Handler

	twitchConfig  twitch.Config
	twitchClient  *helix.Client
	channelUserId string

	q *queries.Queries

	chat           *chat.Client
	chatEventsChan <-chan *chat.Event
	chatEvents     *subcriberChannels[*chat.Event]

	eventHandler *events.Handler
	alerts       *subcriberChannels[*alerts.Alert]
}

func New(ctx context.Context, twitchConfig twitch.Config, twitchClient *helix.Client, channelUserId string, q *queries.Queries, chatClient *chat.Client, chatEventsChan chan *chat.Event) *Server {
	alertsChan := make(chan *alerts.Alert, 32)
	s := &Server{
		twitchConfig:   twitchConfig,
		twitchClient:   twitchClient,
		channelUserId:  channelUserId,
		q:              q,
		chat:           chatClient,
		chatEventsChan: chatEventsChan,
		chatEvents: &subcriberChannels[*chat.Event]{
			chs: make(map[int]chan *chat.Event),
		},
		eventHandler: events.NewHandler(ctx, q, alertsChan),
		alerts: &subcriberChannels[*alerts.Alert]{
			chs: make(map[int]chan *alerts.Alert),
		},
	}

	withCors := cors.New(cors.Options{
		AllowedOrigins: []string{
			"https://localhost:8080",
			fmt.Sprintf("https://%s.ext-twitch.tv", s.twitchConfig.ExtensionClientId),
		},
		AllowedMethods: []string{http.MethodGet},
	})

	r := mux.NewRouter()
	r.Path("/").Methods("GET").HandlerFunc(s.handleStatus)

	authServer := auth.NewServer(channelUserId, twitchConfig.ClientId, twitchConfig.ClientSecret, q)
	authServer.RegisterRoutes(r.PathPrefix("/auth/").Subrouter())

	r.Path("/callback").Methods("POST").HandlerFunc(s.handlePostCallback)
	r.Path("/alerts").Methods("GET").HandlerFunc(s.handleAlerts)
	r.Path("/chat").Methods("GET").HandlerFunc(s.handleChat)
	r.Path("/view").Methods("GET").HandlerFunc(s.handleView)
	s.Handler = withCors.Handler(r)

	go func() {
		for {
			select {
			case <-ctx.Done():
				break
			case event := <-s.chatEventsChan:
				s.chatEvents.broadcast(event)
			case alert := <-alertsChan:
				s.alerts.broadcast(alert)
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
