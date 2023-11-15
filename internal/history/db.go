package history

import "github.com/google/uuid"

// imageRequestSummary is the JSON format used by the GetScreeningsByBroadcastId to
// represent the image requests that occurred during a particular screening
type imageRequestSummary struct {
	Id           uuid.UUID `json:"id"`
	TwitchUserId string    `json:"twitch_user_id"`
	Subject      string    `json:"subject"`
}
