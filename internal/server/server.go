package server

import (
	"net/http"
)

type Server struct {
	http.Handler

	webhookSecret string
}

func New(webhookSecret string) *Server {
	s := &Server{
		webhookSecret: webhookSecret,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleStatus)
	mux.HandleFunc("/callback", s.handlePostCallback)
	s.Handler = mux
	return s
}
