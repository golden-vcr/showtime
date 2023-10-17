package broadcast

import (
	"encoding/json"
	"time"
)

// changeEventNotifyChannel is the name of the NOTIFY channel which the database uses to
// emit events about relevant changes that occur during a broadcast
const changeEventNotifyChannel = "showtime"

// EventType identifies the database table which has been changed via insert or update
type EventType string

const (
	EventTypeBroadcast EventType = "broadcast"
	EventTypeScreening EventType = "screening"
)

// ChangeEvent is a JSON-encoded payload emitted via ChangeEventNotifyChannel
type ChangeEvent struct {
	Type EventType       `json:"type"`
	Data json.RawMessage `json:"data"`
}

// BroadcastEventData is the data for a ChangeEvent of type 'broadcast', representing an
// insert or update in the showtime.broadcast table
type BroadcastEventData struct {
	Id        int        `json:"id"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

// ScreeningEventData is the data for a ChangeEvent of type 'screening', representing an
// insert or update in the showtime.screening table
type ScreeningEventData struct {
	BroadcastId int        `json:"broadcast_id"`
	TapeId      int        `json:"tape_id"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at"`
}
