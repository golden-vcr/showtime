package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/nicklaw5/helix/v2"
)

func (h *Handler) handleChannelFollowEvent(ctx context.Context, data json.RawMessage) error {
	var ev helix.EventSubChannelFollowEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelFollowEvent: %w", err)
	}

	err := h.q.RecordViewerFollow(ctx, queries.RecordViewerFollowParams{
		TwitchUserID:      ev.UserID,
		TwitchDisplayName: ev.UserName,
	})
	if err != nil {
		return fmt.Errorf("RecordViewerFollow failed: %w\n", err)
	}

	h.alertsChan <- &alerts.Alert{
		Type: alerts.AlertTypeFollow,
		Data: alerts.AlertData{
			Follow: &alerts.AlertDataFollow{
				Username: ev.UserName,
			},
		},
	}
	return nil
}

func (h *Handler) handleChannelRaidEvent(ctx context.Context, data json.RawMessage) error {
	var ev helix.EventSubChannelRaidEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelRaidEvent: %w", err)
	}

	h.alertsChan <- &alerts.Alert{
		Type: alerts.AlertTypeRaid,
		Data: alerts.AlertData{
			Raid: &alerts.AlertDataRaid{
				Username:   ev.FromBroadcasterUserName,
				NumViewers: ev.Viewers,
			},
		},
	}
	return nil
}

func (h *Handler) handleChannelCheerEvent(ctx context.Context, data json.RawMessage) error {
	var ev helix.EventSubChannelCheerEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		return fmt.Errorf("failed to unmarshal ChannelCheerEvent: %w", err)
	}

	if !ev.IsAnonymous {
		_, err := h.ledgerClient.RequestCreditFromCheer(ctx, ev.UserID, ev.Bits, ev.Message)
		if err != nil {
			return fmt.Errorf("RequestCreditFromCheer failed: %v", err)
		}
	}
	return nil
}
