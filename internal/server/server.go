package server

import (
	"net/http"

	"github.com/golden-vcr/showtime/internal/eventsub"
)

type Server struct {
	http.Handler

	eventsub      *eventsub.Client
	webhookSecret string
}

func New(eventsubClient *eventsub.Client, webhookSecret string) *Server {
	s := &Server{
		eventsub:      eventsubClient,
		webhookSecret: webhookSecret,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleStatus)
	mux.HandleFunc("/callback", s.handlePostCallback)
	s.Handler = mux
	return s
}
