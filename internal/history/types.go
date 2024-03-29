package history

import (
	"context"
	"time"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	GetBroadcastHistory(ctx context.Context) ([]queries.GetBroadcastHistoryRow, error)
	GetBroadcastById(ctx context.Context, broadcastID int32) (queries.ShowtimeBroadcast, error)
	GetScreeningsByBroadcastId(ctx context.Context, broadcastID int32) ([]queries.GetScreeningsByBroadcastIdRow, error)
	GetViewerLookupForBroadcast(ctx context.Context, broadcastID int32) ([]queries.GetViewerLookupForBroadcastRow, error)
	GetImagesForRequest(ctx context.Context, imageRequestID uuid.UUID) ([]string, error)
}

type Summary struct {
	Broadcasts           []SummarizedBroadcast `json:"broadcasts"`
	BroadcastIdsByTapeId map[string][]int      `json:"broadcastIdsByTapeId"`
}

type SummarizedBroadcast struct {
	Id        int       `json:"id"`
	StartedAt time.Time `json:"startedAt"`
	VodUrl    string    `json:"vodUrl"`
	TapeIds   []int     `json:"tapeIds"`
}

type Broadcast struct {
	Id         int         `json:"id"`
	StartedAt  time.Time   `json:"startedAt"`
	EndedAt    *time.Time  `json:"endedAt"`
	Screenings []Screening `json:"screenings"`
	VodUrl     string      `json:"vodUrl,omitempty"`
}

type Screening struct {
	TapeId        int            `json:"tapeId"`
	StartedAt     time.Time      `json:"startedAt"`
	EndedAt       *time.Time     `json:"endedAt"`
	ImageRequests []ImageRequest `json:"imageRequests"`
}

type ImageRequest struct {
	Id       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Subject  string    `json:"subject"`
}
