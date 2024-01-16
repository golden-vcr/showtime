package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func Test_Server_handleGetSummary(t *testing.T) {
	tests := []struct {
		name       string
		q          *mockQueries
		wantStatus int
		wantBody   string
	}{
		{
			"result is empty when no data exists",
			&mockQueries{},
			http.StatusOK,
			`{"broadcastIdsByTapeId":{}}`,
		},
		{
			"summary correlates tape IDs to broadcasts in which tapes were screened",
			&mockQueries{
				broadcasts: []mockBroadcast{
					{
						id:        1,
						startedAt: time.Now().Add(-12 * time.Hour),
						endedAt:   sql.NullTime{Valid: true, Time: time.Now().Add(-10 * time.Hour)},
					},
					{
						id:        2,
						startedAt: time.Now().Add(-6 * time.Hour),
						endedAt:   sql.NullTime{Valid: true, Time: time.Now().Add(-4 * time.Hour)},
					},
					{
						id:        3,
						startedAt: time.Now().Add(-1 * time.Hour),
					},
				},
				screenings: []mockScreening{
					{
						broadcastId: 1,
						tapeId:      44,
					},
					{
						broadcastId: 1,
						tapeId:      22,
					},
					{
						broadcastId: 2,
						tapeId:      66,
					},
					{
						broadcastId: 3,
						tapeId:      44,
					},
					{
						broadcastId: 3,
						tapeId:      11,
					},
				},
			},
			http.StatusOK,
			`{"broadcastIdsByTapeId":{"11":[3],"22":[1],"44":[1,3],"66":[2]}}`,
		},
		{
			"database error is a 500",
			&mockQueries{
				err: fmt.Errorf("mock error"),
			},
			http.StatusInternalServerError,
			"mock error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Server{q: tt.q}
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			res := httptest.NewRecorder()
			s.handleGetSummary(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func Test_Server_handleGetBroadcast(t *testing.T) {
	tests := []struct {
		name        string
		q           *mockQueries
		broadcastId int
		wantStatus  int
		wantBody    string
	}{
		{
			"normal usage",
			&mockQueries{
				broadcasts: []mockBroadcast{
					{
						id:        1,
						startedAt: time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						endedAt:   sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 14, 0, 0, 0, time.UTC)},
					},
				},
				screenings: []mockScreening{
					{
						broadcastId: 1,
						tapeId:      44,
						startedAt:   time.Date(1997, 9, 1, 12, 15, 0, 0, time.UTC),
						endedAt:     sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 45, 0, 0, time.UTC)},
					},
					{
						broadcastId: 1,
						tapeId:      22,
						startedAt:   time.Date(1997, 9, 1, 12, 55, 0, 0, time.UTC),
						endedAt:     sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 13, 30, 0, 0, time.UTC)},
					},
				},
			},
			1,
			http.StatusOK,
			`{"id":1,"startedAt":"1997-09-01T12:00:00Z","endedAt":"1997-09-01T14:00:00Z","screenings":[{"tapeId":44,"startedAt":"1997-09-01T12:15:00Z","endedAt":"1997-09-01T12:45:00Z","imageRequests":[]},{"tapeId":22,"startedAt":"1997-09-01T12:55:00Z","endedAt":"1997-09-01T13:30:00Z","imageRequests":[]}]}`,
		},
		{
			"image requests made during each screening are reported",
			&mockQueries{
				broadcasts: []mockBroadcast{
					{
						id:        1,
						startedAt: time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						endedAt:   sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 14, 0, 0, 0, time.UTC)},
					},
				},
				screenings: []mockScreening{
					{
						broadcastId: 1,
						tapeId:      44,
						startedAt:   time.Date(1997, 9, 1, 12, 15, 0, 0, time.UTC),
						endedAt:     sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 45, 0, 0, time.UTC)},
						imageRequests: []imageRequestSummary{
							{
								Id:           uuid.MustParse("4c511c13-e4e6-48eb-94e2-45beed2fd11c"),
								TwitchUserId: "1234",
								Subject:      "a big rock",
							},
						},
					},
				},
			},
			1,
			http.StatusOK,
			`{"id":1,"startedAt":"1997-09-01T12:00:00Z","endedAt":"1997-09-01T14:00:00Z","screenings":[{"tapeId":44,"startedAt":"1997-09-01T12:15:00Z","endedAt":"1997-09-01T12:45:00Z","imageRequests":[{"id":"4c511c13-e4e6-48eb-94e2-45beed2fd11c","username":"User 1234","subject":"a big rock"}]}]}`,
		},
		{
			"twitch display names are resolved from  user ids for image requests",
			&mockQueries{
				broadcasts: []mockBroadcast{
					{
						id:        1,
						startedAt: time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						endedAt:   sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 14, 0, 0, 0, time.UTC)},
					},
				},
				screenings: []mockScreening{
					{
						broadcastId: 1,
						tapeId:      44,
						startedAt:   time.Date(1997, 9, 1, 12, 15, 0, 0, time.UTC),
						endedAt:     sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 12, 45, 0, 0, time.UTC)},
						imageRequests: []imageRequestSummary{
							{
								Id:           uuid.MustParse("4c511c13-e4e6-48eb-94e2-45beed2fd11c"),
								TwitchUserId: "1234",
								Subject:      "a big rock",
							},
						},
					},
				},
				viewerLookupRows: []queries.GetViewerLookupForBroadcastRow{
					{
						TwitchUserID:      "1234",
						TwitchDisplayName: "PersonMan",
					},
				},
			},
			1,
			http.StatusOK,
			`{"id":1,"startedAt":"1997-09-01T12:00:00Z","endedAt":"1997-09-01T14:00:00Z","screenings":[{"tapeId":44,"startedAt":"1997-09-01T12:15:00Z","endedAt":"1997-09-01T12:45:00Z","imageRequests":[{"id":"4c511c13-e4e6-48eb-94e2-45beed2fd11c","username":"PersonMan","subject":"a big rock"}]}]}`,
		},
		{
			"if screening end time is invalid, broadcast end time is substituted",
			&mockQueries{
				broadcasts: []mockBroadcast{
					{
						id:        1,
						startedAt: time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
						endedAt:   sql.NullTime{Valid: true, Time: time.Date(1997, 9, 1, 14, 0, 0, 0, time.UTC)},
					},
				},
				screenings: []mockScreening{
					{
						broadcastId: 1,
						tapeId:      44,
						startedAt:   time.Date(1997, 9, 1, 12, 15, 0, 0, time.UTC),
					},
				},
			},
			1,
			http.StatusOK,
			`{"id":1,"startedAt":"1997-09-01T12:00:00Z","endedAt":"1997-09-01T14:00:00Z","screenings":[{"tapeId":44,"startedAt":"1997-09-01T12:15:00Z","endedAt":"1997-09-01T14:00:00Z","imageRequests":[]}]}`,
		},
		{
			"if broadcast is in progress, Broadcast.endedAt is null and Screening.endedAt may be null as well",
			&mockQueries{
				broadcasts: []mockBroadcast{
					{
						id:        1,
						startedAt: time.Date(1997, 9, 1, 12, 0, 0, 0, time.UTC),
					},
				},
				screenings: []mockScreening{
					{
						broadcastId: 1,
						tapeId:      44,
						startedAt:   time.Date(1997, 9, 1, 12, 15, 0, 0, time.UTC),
					},
				},
			},
			1,
			http.StatusOK,
			`{"id":1,"startedAt":"1997-09-01T12:00:00Z","endedAt":null,"screenings":[{"tapeId":44,"startedAt":"1997-09-01T12:15:00Z","endedAt":null,"imageRequests":[]}]}`,
		},
		{
			"invalid broadcast ID is a 404",
			&mockQueries{},
			1,
			http.StatusNotFound,
			"no such broadcast",
		},
		{
			"database error is a 500",
			&mockQueries{
				err: fmt.Errorf("mock error"),
			},
			1,
			http.StatusInternalServerError,
			"mock error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Server{q: tt.q}
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%d", tt.broadcastId), nil)
			req = mux.SetURLVars(req, map[string]string{"id": fmt.Sprintf("%d", tt.broadcastId)})
			res := httptest.NewRecorder()
			s.handleGetBroadcast(res, req)

			b, err := io.ReadAll(res.Body)
			assert.NoError(t, err)
			body := strings.TrimSuffix(string(b), "\n")
			assert.Equal(t, tt.wantStatus, res.Code)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

type mockQueries struct {
	err              error
	broadcasts       []mockBroadcast
	screenings       []mockScreening
	viewerLookupRows []queries.GetViewerLookupForBroadcastRow
}

type mockBroadcast struct {
	id        int32
	startedAt time.Time
	endedAt   sql.NullTime
}

type mockScreening struct {
	broadcastId   int32
	tapeId        int32
	startedAt     time.Time
	endedAt       sql.NullTime
	imageRequests []imageRequestSummary
}

func (m *mockQueries) GetBroadcastHistory(ctx context.Context) ([]queries.GetBroadcastHistoryRow, error) {
	if m.err != nil {
		return nil, m.err
	}

	rows := make([]queries.GetBroadcastHistoryRow, 0, len(m.broadcasts))
	for _, broadcast := range m.broadcasts {
		tapeIds := make([]int32, 0, 8)
		lastScreeningStartedAt := time.Time{}
		for _, screening := range m.screenings {
			if screening.broadcastId == broadcast.id {
				if screening.startedAt.Before(lastScreeningStartedAt) {
					return nil, fmt.Errorf("test invariant failed: screenings should be listed in order based on start time")
				}
				tapeIds = append(tapeIds, screening.tapeId)
				lastScreeningStartedAt = screening.startedAt
			}
		}
		rows = append(rows, queries.GetBroadcastHistoryRow{
			ID:        broadcast.id,
			StartedAt: broadcast.startedAt,
			TapeIds:   tapeIds,
		})
	}
	return rows, nil
}

func (m *mockQueries) GetTapeScreeningHistory(ctx context.Context) ([]queries.GetTapeScreeningHistoryRow, error) {
	if m.err != nil {
		return nil, m.err
	}

	broadcastIdsByTapeId := make(map[int32]map[int32]struct{})
	for _, s := range m.screenings {
		if broadcastIdsByTapeId[s.tapeId] == nil {
			broadcastIdsByTapeId[s.tapeId] = make(map[int32]struct{})
		}
		broadcastIdsByTapeId[s.tapeId][s.broadcastId] = struct{}{}
	}

	tapeIds := make([]int32, 0, len(broadcastIdsByTapeId))
	for tapeId := range broadcastIdsByTapeId {
		tapeIds = append(tapeIds, tapeId)
	}
	sort.Slice(tapeIds, func(i, j int) bool { return tapeIds[i] < tapeIds[j] })

	rows := make([]queries.GetTapeScreeningHistoryRow, 0, len(tapeIds))
	for _, tapeId := range tapeIds {
		broadcastIds := make([]int32, 0, len(broadcastIdsByTapeId[tapeId]))
		for broadcastId := range broadcastIdsByTapeId[tapeId] {
			broadcastIds = append(broadcastIds, broadcastId)
		}
		sort.Slice(broadcastIds, func(i, j int) bool { return broadcastIds[i] < broadcastIds[j] })
		rows = append(rows, queries.GetTapeScreeningHistoryRow{
			TapeID:       tapeId,
			BroadcastIds: broadcastIds,
		})
	}
	return rows, nil
}

func (m *mockQueries) GetBroadcastById(ctx context.Context, broadcastID int32) (queries.ShowtimeBroadcast, error) {
	if m.err != nil {
		return queries.ShowtimeBroadcast{}, m.err
	}
	for _, b := range m.broadcasts {
		if b.id == broadcastID {
			return queries.ShowtimeBroadcast{
				ID:        b.id,
				StartedAt: b.startedAt,
				EndedAt:   b.endedAt,
			}, nil
		}
	}
	return queries.ShowtimeBroadcast{}, sql.ErrNoRows
}

func (m *mockQueries) GetScreeningsByBroadcastId(ctx context.Context, broadcastID int32) ([]queries.GetScreeningsByBroadcastIdRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	rows := make([]queries.GetScreeningsByBroadcastIdRow, 0)
	for _, s := range m.screenings {
		if s.broadcastId == broadcastID {
			summaries, err := json.Marshal(s.imageRequests)
			if err != nil {
				return nil, err
			}
			rows = append(rows, queries.GetScreeningsByBroadcastIdRow{
				TapeID:        s.tapeId,
				StartedAt:     s.startedAt,
				EndedAt:       s.endedAt,
				ImageRequests: summaries,
			})
		}
	}
	return rows, nil
}

func (m *mockQueries) GetViewerLookupForBroadcast(ctx context.Context, broadcastID int32) ([]queries.GetViewerLookupForBroadcastRow, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.viewerLookupRows, nil
}

func (m *mockQueries) GetImagesForRequest(ctx context.Context, imageRequestID uuid.UUID) ([]string, error) {
	return nil, nil
}

var _ Queries = (*mockQueries)(nil)
