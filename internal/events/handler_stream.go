package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golden-vcr/showtime/gen/queries"
)

func (h *Handler) handleStreamOnlineEvent(ctx context.Context, data json.RawMessage) error {
	// Check the most recent broadcast to see if it ended very recently
	broadcast, err := getMostRecentBroadcast(ctx, h.q)
	if err != nil {
		return fmt.Errorf("error getting most recent broadcast: %w", err)
	}
	if broadcast != nil {
		if broadcast.EndedAt.Valid {
			// If the broadcast ended recently, resume it and we're done
			fifteenMinutesAgo := time.Now().Add(-15 * time.Minute)
			if broadcast.EndedAt.Time.After(fifteenMinutesAgo) {
				if err := h.q.RecordBroadcastResumed(ctx, broadcast.ID); err != nil {
					return fmt.Errorf("error resuming broadcast %d: %w", broadcast.ID, err)
				}
				fmt.Printf("[BROADCAST %d] Stream has come back online; broadcast is resumed.\n", broadcast.ID)
				return nil
			}
		} else {
			// If the broadcast is still showing up as live, log a warning and continue
			fmt.Printf("WARNING: received stream.online event while still purportedly live (last broadcast was %d, started at %s)\n", broadcast.ID, broadcast.StartedAt.Format(time.RFC3339Nano))
		}
	}

	// We either don't have a previous broadcast, the previous broadcast ended more than
	// 15 minutes ago, or the previous broadcast is still erroneously showing up as
	// live: either way, we want to start a new broadcast
	newBroadcastId, err := h.q.RecordBroadcastStarted(ctx)
	if err != nil {
		return fmt.Errorf("error recording start of broadcast: %w", err)
	}
	fmt.Printf("[BROADCAST %d] Stream has come online; broadcast is started.\n", newBroadcastId)
	return nil
}

func (h *Handler) handleStreamOfflineEvent(ctx context.Context, data json.RawMessage) error {
	// Check the most recent broadcast to see if it's still live
	broadcast, err := getMostRecentBroadcast(ctx, h.q)
	if err != nil {
		return fmt.Errorf("error getting most recent broadcast: %w", err)
	}

	// If we have no recent broadcast, or if the most recent broadcast is already ended,
	// silently accept the event without doing anything, but print a warning
	if broadcast == nil || broadcast.EndedAt.Valid {
		suffix := ""
		if broadcast != nil {
			suffix = fmt.Sprintf(" (last broadcast was %d, ended at %s)", broadcast.ID, broadcast.EndedAt.Time.Format(time.RFC3339Nano))
		}
		fmt.Printf("WARNING: received stream.offline event with no active broadcast%s\n", suffix)
		return nil
	}

	// Flag the broadcast as ended as of right now
	if err := h.q.RecordBroadcastEnded(ctx); err != nil {
		return fmt.Errorf("error recording end of broadcast: %w", err)
	}
	fmt.Printf("[BROADCAST %d] Stream has gone offline; broadcast is ended.\n", broadcast.ID)
	return nil
}

func getMostRecentBroadcast(ctx context.Context, q *queries.Queries) (*queries.GetMostRecentBroadcastRow, error) {
	result, err := q.GetMostRecentBroadcast(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}
