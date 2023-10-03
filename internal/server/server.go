package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nicklaw5/helix/v2"
	"github.com/rs/cors"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/golden-vcr/showtime/internal/chat"
	"github.com/golden-vcr/showtime/internal/events"
	"github.com/golden-vcr/showtime/internal/sse"
	"github.com/golden-vcr/showtime/internal/twitch"
)

type Server struct {
	http.Handler

	twitchConfig  twitch.Config
	twitchClient  *helix.Client
	channelUserId string

	q *queries.Queries

	chatAgent    *chat.Agent
	eventHandler *events.Handler
}

func New(ctx context.Context, twitchConfig twitch.Config, twitchClient *helix.Client, channelUserId string, q *queries.Queries, chatAgent *chat.Agent) *Server {
	alertsChan := make(chan *alerts.Alert, 32)
	s := &Server{
		twitchConfig:  twitchConfig,
		twitchClient:  twitchClient,
		channelUserId: channelUserId,
		q:             q,
		chatAgent:     chatAgent,
		eventHandler:  events.NewHandler(ctx, q, alertsChan),
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

	eventsHandler := events.NewHandler(context.Background(), q, alertsChan)
	eventsServer := events.NewServer(twitchConfig.WebhookSecret, eventsHandler)
	r.Path("/callback").Methods("POST").Handler(eventsServer)

	chatHandler := sse.NewHandler[*chat.LogEvent](ctx, s.chatAgent.GetLogEvents())
	r.Path("/chat").Methods("GET").Handler(chatHandler)

	alertsHandler := sse.NewHandler[*alerts.Alert](ctx, alertsChan)
	r.Path("/alerts").Methods("GET").Handler(alertsHandler)

	r.Path("/view").Methods("GET").HandlerFunc(s.handleView)
	s.Handler = withCors.Handler(r)

	return s
}
