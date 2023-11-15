package history

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/gorilla/mux"
)

type Server struct {
	q Queries
}

func NewServer(q *queries.Queries) *Server {
	return &Server{
		q: q,
	}
}

func (s *Server) RegisterRoutes(r *mux.Router) {
	for _, root := range []string{"", "/"} {
		r.Path(root).Methods("GET").HandlerFunc(s.handleGetSummary)
	}
	r.Path("/{id}").Methods("GET").HandlerFunc(s.handleGetBroadcast)
}

func (s *Server) handleGetSummary(res http.ResponseWriter, req *http.Request) {
	// Get a row for each tape that's ever been screened, along with the set of
	// broadcast IDs in which that tape was screened
	rows, err := s.q.GetTapeScreeningHistory(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a summary to return to the client as JSON
	broadcastIdsByTapeId := make(map[string][]int)
	for _, row := range rows {
		tapeIdStr := fmt.Sprintf("%d", row.TapeID)
		broadcastIds := make([]int, 0, len(row.BroadcastIds))
		for _, broadcastId := range row.BroadcastIds {
			broadcastIds = append(broadcastIds, int(broadcastId))
		}
		broadcastIdsByTapeId[tapeIdStr] = broadcastIds
	}
	summary := Summary{BroadcastIdsByTapeId: broadcastIdsByTapeId}
	if err := json.NewEncoder(res).Encode(summary); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGetBroadcast(res http.ResponseWriter, req *http.Request) {
	// Figure out which broadcast we want to get history for
	broadcastIdStr, ok := mux.Vars(req)["id"]
	if !ok || broadcastIdStr == "" {
		http.Error(res, "failed to parse 'id' from URL", http.StatusInternalServerError)
		return
	}
	broadcastId, err := strconv.Atoi(broadcastIdStr)
	if err != nil {
		http.Error(res, "broadcast ID must be an integer", http.StatusBadRequest)
		return
	}

	// Ensure that a broadcastRow exists with that ID
	broadcastRow, err := s.q.GetBroadcastById(req.Context(), int32(broadcastId))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(res, "no such broadcast", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Find all screeningRows recorded within that broadcast
	screeningRows, err := s.q.GetScreeningsByBroadcastId(req.Context(), broadcastRow.ID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a result struct and return it as JSON
	screenings := make([]Screening, 0, len(screeningRows))
	for i := range screeningRows {
		// If the screening does not have a valid end time but the broadcast itself is
		// ended, inherit the end time of the broadcast: otherwise a null ended-at time
		// indicates that the screening is in progress
		var screeningEndedAt *time.Time
		if screeningRows[i].EndedAt.Valid {
			screeningEndedAt = &screeningRows[i].EndedAt.Time
		} else if broadcastRow.EndedAt.Valid {
			screeningEndedAt = &broadcastRow.EndedAt.Time
		}

		// The 'image_requests' JSON payload should be an array of objects that conform
		// to imageRequestSummary
		var summaries []imageRequestSummary
		if err := json.Unmarshal(screeningRows[i].ImageRequests, &summaries); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		imageRequests := make([]ImageRequest, 0, len(summaries))
		for _, summary := range summaries {
			imageRequests = append(imageRequests, ImageRequest{
				Id:       summary.Id,
				Username: fmt.Sprintf("User %s", summary.TwitchUserId),
				Subject:  summary.Subject,
			})
		}

		screenings = append(screenings, Screening{
			TapeId:        int(screeningRows[i].TapeID),
			StartedAt:     screeningRows[i].StartedAt,
			EndedAt:       screeningEndedAt,
			ImageRequests: imageRequests,
		})
	}
	var broadcastEndedAt *time.Time
	if broadcastRow.EndedAt.Valid {
		broadcastEndedAt = &broadcastRow.EndedAt.Time
	}
	broadcast := Broadcast{
		Id:         int(broadcastRow.ID),
		StartedAt:  broadcastRow.StartedAt,
		EndedAt:    broadcastEndedAt,
		Screenings: screenings,
	}
	if err := json.NewEncoder(res).Encode(broadcast); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
