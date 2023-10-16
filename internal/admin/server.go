package admin

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

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
	// Figure out which tape we want to screen
	tapeIdStr, ok := mux.Vars(req)["id"]
	if !ok || tapeIdStr == "" {
		http.Error(res, "failed to parse 'id' from URL", http.StatusInternalServerError)
		return
	}
	tapeId, err := strconv.Atoi(tapeIdStr)
	if err != nil {
		http.Error(res, "tape ID must be an integer", http.StatusBadRequest)
		return
	}

	// Resolve the ID of the current broadcast, if it's live (i.e. not ended)
	broadcast, err := s.q.GetMostRecentBroadcast(req.Context())
	if errors.Is(err, sql.ErrNoRows) || (err == nil && broadcast.EndedAt.Valid) {
		http.Error(res, "no broadcast is currently live", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a screening record for this tape in this broadcast
	if err := s.q.RecordScreeningStarted(req.Context(), queries.RecordScreeningStartedParams{
		BroadcastID: broadcast.ID,
		TapeID:      int32(tapeId),
	}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleClearTape(res http.ResponseWriter, req *http.Request) {
	// Resolve the ID of the current broadcast, if it's live (i.e. not ended)
	broadcast, err := s.q.GetMostRecentBroadcast(req.Context())
	if errors.Is(err, sql.ErrNoRows) || (err == nil && broadcast.EndedAt.Valid) {
		http.Error(res, "no broadcast is currently live", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set ended_at timestamps on all screening records for that broadcast that are not
	// already ended
	if err := s.q.RecordScreeningEnded(req.Context(), broadcast.ID); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusNoContent)
}
