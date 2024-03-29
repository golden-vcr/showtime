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
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
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
	r.Path("/images/{id}").Methods("GET").HandlerFunc(s.handleGetImages)
}

func (s *Server) handleGetSummary(res http.ResponseWriter, req *http.Request) {
	// Get a row for each broadcast in which we've screened any tapes, including which
	// tape IDs were screened in which broadcasts
	rows, err := s.q.GetBroadcastHistory(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build a list containing the summarized details of each past broadcast
	broadcasts := make([]SummarizedBroadcast, 0, len(rows))
	for _, row := range rows {
		vodUrl := ""
		if row.VodUrl.Valid {
			vodUrl = row.VodUrl.String
		}
		tapeIds := make([]int, 0, len(row.TapeIds))
		for _, tapeId := range row.TapeIds {
			tapeIds = append(tapeIds, int(tapeId))
		}
		broadcasts = append(broadcasts, SummarizedBroadcast{
			Id:        int(row.ID),
			StartedAt: row.StartedAt,
			VodUrl:    vodUrl,
			TapeIds:   tapeIds,
		})
	}

	// Build a reverse lookup which allows the client to look up which broadcasts a
	// specific tape has been screened in
	broadcastIdsByTapeId := make(map[string][]int)
	for _, broadcast := range rows {
		for _, tapeId := range broadcast.TapeIds {
			tapeIdStr := fmt.Sprintf("%d", tapeId)
			existingBroadcastIds, ok := broadcastIdsByTapeId[tapeIdStr]
			if ok {
				broadcastIdsByTapeId[tapeIdStr] = append(existingBroadcastIds, int(broadcast.ID))
			} else {
				broadcastIds := make([]int, 0, 8)
				broadcastIds = append(broadcastIds, int(broadcast.ID))
				broadcastIdsByTapeId[tapeIdStr] = broadcastIds
			}
		}
	}
	summary := Summary{
		Broadcasts:           broadcasts,
		BroadcastIdsByTapeId: broadcastIdsByTapeId,
	}
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

	// Run two queries concurrently to get the data we need for this request: all the
	// screenings recorded within that broadcast (including image request summaries
	// etc.), along with a lookup that maps Twitch User IDs to display names
	screeningsChan := make(chan []queries.GetScreeningsByBroadcastIdRow, 1)
	viewerLookupChan := make(chan []queries.GetViewerLookupForBroadcastRow, 1)
	wg, queryCtx := errgroup.WithContext(req.Context())
	wg.Go(func() error {
		// Find all screening rows recorded within the broadcast
		screeningRows, err := s.q.GetScreeningsByBroadcastId(queryCtx, broadcastRow.ID)
		if err != nil {
			return err
		}
		screeningsChan <- screeningRows
		return nil
	})
	wg.Go(func() error {
		// Get a lookup of twitch user IDs to names
		viewerLookupRows, err := s.q.GetViewerLookupForBroadcast(queryCtx, broadcastRow.ID)
		if err != nil {
			return err
		}
		viewerLookupChan <- viewerLookupRows
		return nil
	})
	if err := wg.Wait(); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	screeningRows := <-screeningsChan
	viewerLookupRows := <-viewerLookupChan

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
				Username: formatUsername(viewerLookupRows, summary.TwitchUserId),
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
	vodUrl := ""
	if broadcastRow.VodUrl.Valid {
		vodUrl = broadcastRow.VodUrl.String
	}
	broadcast := Broadcast{
		Id:         int(broadcastRow.ID),
		StartedAt:  broadcastRow.StartedAt,
		EndedAt:    broadcastEndedAt,
		Screenings: screenings,
		VodUrl:     vodUrl,
	}
	if err := json.NewEncoder(res).Encode(broadcast); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleGetImages(res http.ResponseWriter, req *http.Request) {
	// Figure out which image request we want to get image URLs for
	requestIdStr, ok := mux.Vars(req)["id"]
	if !ok || requestIdStr == "" {
		http.Error(res, "failed to parse 'id' from URL", http.StatusInternalServerError)
		return
	}
	requestId, err := uuid.Parse(requestIdStr)
	if err != nil {
		http.Error(res, "image request ID must be a uuid", http.StatusBadRequest)
		return
	}

	imageUrls, err := s.q.GetImagesForRequest(req.Context(), requestId)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(res).Encode(imageUrls); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

func formatUsername(viewerLookupRows []queries.GetViewerLookupForBroadcastRow, twitchUserId string) string {
	for _, row := range viewerLookupRows {
		if row.TwitchUserID == twitchUserId {
			return row.TwitchDisplayName
		}
	}
	return fmt.Sprintf("User %s", twitchUserId)
}
