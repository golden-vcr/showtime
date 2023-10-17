package broadcast

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type ChangeListener struct {
	pql *pq.Listener

	lastKnownBroadcastId        int
	lastKnownBroadcastStartedAt time.Time
	lastKnownScreeningStartedAt time.Time

	state State
}

func NewChangeListener(ctx context.Context, pql *pq.Listener, q Queries) (*ChangeListener, error) {
	err := pql.Listen(changeEventNotifyChannel)
	if err != nil {
		return nil, err
	}
	l := &ChangeListener{pql: pql}
	if err := l.initialize(ctx, q); err != nil {
		return nil, err
	}
	fmt.Printf("STATE INIT: %+v\n", l.state)
	return l, nil
}

func (l *ChangeListener) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return l.pql.Close()
		case notification := <-l.pql.Notify:
			if notification.Channel == changeEventNotifyChannel {
				var event ChangeEvent
				if err := json.Unmarshal([]byte(notification.Extra), &event); err != nil {
					return fmt.Errorf("failed to decode JSON payload from pg event in channel '%s': %w", notification.Channel, err)
				}
				switch event.Type {
				case EventTypeBroadcast:
					{
						var data BroadcastEventData
						if err := json.Unmarshal(event.Data, &data); err != nil {
							return fmt.Errorf("failed to decode JSON data for '%s' event in channel '%s': %w", event.Type, notification.Channel, err)
						}
						l.handleBroadcastChange(&data)
					}
				case EventTypeScreening:
					{
						var data ScreeningEventData
						if err := json.Unmarshal(event.Data, &data); err != nil {
							return fmt.Errorf("failed to decode JSON data for '%s' event in channel '%s': %w", event.Type, notification.Channel, err)
						}
						l.handleScreeningChange(&data)
					}
				default:
					return fmt.Errorf("unrecognized event type '%s' in channel '%s'", event.Type, notification.Channel)
				}
			}
		}
	}
}

func (l *ChangeListener) initialize(ctx context.Context, q Queries) error {
	broadcast, err := q.GetMostRecentBroadcast(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		l.lastKnownBroadcastId = int(broadcast.ID)
		l.lastKnownBroadcastStartedAt = broadcast.StartedAt
		if !broadcast.EndedAt.Valid {
			l.state.IsLive = true
			l.state.BroadcastStartedAt = broadcast.StartedAt
		}

		screening, err := q.GetMostRecentScreening(ctx, broadcast.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err == nil {
			l.lastKnownScreeningStartedAt = screening.StartedAt
			if !screening.EndedAt.Valid {
				l.state.ScreeningTapeId = int(screening.TapeID)
				l.state.ScreeningStartedAt = screening.StartedAt
			}
		}
	}
	return nil
}

func (l *ChangeListener) handleBroadcastChange(data *BroadcastEventData) {
	if data.StartedAt.Before(l.lastKnownBroadcastStartedAt) {
		return
	}
	l.lastKnownBroadcastId = data.Id
	l.lastKnownBroadcastStartedAt = data.StartedAt

	if data.EndedAt != nil {
		l.updateState(&State{IsLive: false})
	} else {
		l.updateState(&State{
			IsLive:             true,
			BroadcastStartedAt: data.StartedAt,
		})
	}
}

func (l *ChangeListener) handleScreeningChange(data *ScreeningEventData) {
	if data.BroadcastId != l.lastKnownBroadcastId {
		return
	}
	if l.lastKnownBroadcastStartedAt.Equal(time.Time{}) {
		return
	}
	if data.StartedAt.Before(l.lastKnownScreeningStartedAt) {
		return
	}
	l.lastKnownScreeningStartedAt = data.StartedAt

	if data.EndedAt != nil {
		l.updateState(&State{
			IsLive:             true,
			BroadcastStartedAt: l.lastKnownBroadcastStartedAt,
		})
	} else {
		l.updateState(&State{
			IsLive:             true,
			BroadcastStartedAt: l.lastKnownBroadcastStartedAt,
			ScreeningTapeId:    data.TapeId,
			ScreeningStartedAt: data.StartedAt,
		})
	}
}

func (l *ChangeListener) updateState(state *State) {
	fmt.Printf("STATE CHANGE: %+v\n", state)
	l.state = *state
}
