package events

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/nicklaw5/helix/v2"
)

var ErrUnsupportedEventType = errors.New("unsupported event type")

type Handler struct {
	q          *queries.Queries
	alertsChan chan *alerts.Alert
}

func NewHandler(ctx context.Context, q *queries.Queries, alertsChan chan *alerts.Alert) *Handler {
	return &Handler{
		q:          q,
		alertsChan: alertsChan,
	}
}

func (h *Handler) HandleEvent(ctx context.Context, subscription *helix.EventSubSubscription, data json.RawMessage) error {
	switch subscription.Type {
	case helix.EventSubTypeStreamOnline:
		return h.handleStreamOnlineEvent(ctx, data)
	case helix.EventSubTypeStreamOffline:
		return h.handleStreamOfflineEvent(ctx, data)
	case helix.EventSubTypeChannelFollow:
		return h.handleChannelFollowEvent(ctx, data)
	case helix.EventSubTypeChannelRaid:
		return h.handleChannelRaidEvent(ctx, data)
	default:
		return ErrUnsupportedEventType
	}
}
