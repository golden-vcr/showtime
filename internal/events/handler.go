package events

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/ledger"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/nicklaw5/helix/v2"
)

var ErrUnsupportedEventType = errors.New("unsupported event type")

type Handler struct {
	q                 *queries.Queries
	alertsChan        chan *alerts.Alert
	authServiceClient auth.ServiceClient
	ledgerClient      ledger.Client
	imagegenUrl       string
	imagegenCtx       context.Context
}

func NewHandler(ctx context.Context, q *queries.Queries, alertsChan chan *alerts.Alert, authServiceClient auth.ServiceClient, ledgerClient ledger.Client) *Handler {
	return &Handler{
		q:                 q,
		alertsChan:        alertsChan,
		authServiceClient: authServiceClient,
		ledgerClient:      ledgerClient,
		imagegenUrl:       "http://localhost:5001/image-gen",
		imagegenCtx:       ctx,
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
	case helix.EventSubTypeChannelCheer:
		return h.handleChannelCheerEvent(ctx, data)
	case helix.EventSubTypeChannelSubscription:
		return h.handleChannelSubscriptionEvent(ctx, data)
	case helix.EventSubTypeChannelSubscriptionEnd:
		return h.handleChannelSubscriptionEndEvent(ctx, data)
	case helix.EventSubTypeChannelSubscriptionGift:
		return h.handleChannelSubscriptionGiftEvent(ctx, data)
	case helix.EventSubTypeChannelSubscriptionMessage:
		return h.handleChannelSubscriptionMessageEvent(ctx, data)
	default:
		return ErrUnsupportedEventType
	}
}
