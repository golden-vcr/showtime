package broadcast

import (
	"context"
	"time"

	"github.com/golden-vcr/showtime/gen/queries"
)

type Queries interface {
	GetMostRecentBroadcast(ctx context.Context) (queries.GetMostRecentBroadcastRow, error)
	GetMostRecentScreening(ctx context.Context, broadcastID int32) (queries.GetMostRecentScreeningRow, error)
}

// State describes the state of the broadcast that's currently happening, if any
type State struct {
	// IsLive is true if there's currently a live stream happening on the Twitch channel
	IsLive bool `json:"isLive"`
	// BroadcastStartedAt is the UTC timestamp at which the current broadcast started,
	// if a broadcast is live
	BroadcastStartedAt *time.Time `json:"broadcastStartedAt,omitempty"`
	// ScreeningTapeId is the numeric ID of the tape we're currently screening, if any
	ScreeningTapeId int `json:"screeningTapeId,omitempty"`
	// ScreeningStartedAt is the UTC timestamp indicating when we started screening the
	// current tape, if a tape is currently being screened
	ScreeningStartedAt *time.Time `json:"screeningStartedAt,omitempty"`
}
