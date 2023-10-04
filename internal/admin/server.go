package admin

import (
	"net/http"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/gorilla/mux"
)

type Server struct {
	q *queries.Queries
}

func NewServer(q *queries.Queries) *Server {
	return &Server{
		q: q,
	}
}

func (s *Server) RegisterRoutes(c auth.Client, r *mux.Router) {
	// Require broadcaster access for all admin routes
	r.Use(func(next http.Handler) http.Handler {
		return auth.RequireAccess(c, auth.RoleBroadcaster, next)
	})

	// POST /tape allows the broadcaster to notify the backend that we're now screening
	// a new tape
	r.Path("/tape/{id}").Methods("POST").HandlerFunc(s.handleSetTape)
	r.Path("/tape").Methods("DELETE").HandlerFunc(s.handleClearTape)
}

func (s *Server) handleSetTape(res http.ResponseWriter, req *http.Request) {
	tapeId, ok := mux.Vars(req)["id"]
	if !ok || tapeId == "" {
		http.Error(res, "failed to parse 'id' from URL", http.StatusInternalServerError)
		return
	}
	if err := s.q.SetTapeId(req.Context(), tapeId); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleClearTape(res http.ResponseWriter, req *http.Request) {
	if err := s.q.ClearTapeId(req.Context()); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusNoContent)
}
