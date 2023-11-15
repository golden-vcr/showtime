package history

import (
	"context"
	"time"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/google/uuid"
)

type Queries interface {
	GetTapeScreeningHistory(ctx context.Context) ([]queries.GetTapeScreeningHistoryRow, error)
	GetBroadcastById(ctx context.Context, broadcastID int32) (queries.ShowtimeBroadcast, error)
	GetScreeningsByBroadcastId(ctx context.Context, broadcastID int32) ([]queries.GetScreeningsByBroadcastIdRow, error)
}

type Summary struct {
	BroadcastIdsByTapeId map[string][]int `json:"broadcastIdsByTapeId"`
}

type Broadcast struct {
	Id         int         `json:"id"`
	StartedAt  time.Time   `json:"startedAt"`
	EndedAt    *time.Time  `json:"endedAt"`
	Screenings []Screening `json:"screenings"`
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
