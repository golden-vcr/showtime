package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/golden-vcr/auth"
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

	// Anonymous cheers can have no effect in the backend; we can't know who to credit
	if ev.IsAnonymous {
		return nil
	}

	// Contact the auth server to request a short-lived JWT that will give us
	// authoritative access to the backend resources associated with the user who gave
	// us bits
	accessToken, err := h.authServiceClient.RequestServiceToken(ctx, auth.ServiceTokenRequest{
		Service: "showtime",
		User: auth.UserDetails{
			Id:          ev.UserID,
			Login:       ev.UserLogin,
			DisplayName: ev.UserName,
		},
	})
	if err != nil {
		return fmt.Errorf("RequestServiceToken failed: %w", err)
	}

	// Supply that JWT (which is marked as 'authoritative' since it was issued to a
	// trusted internal service) to the ledger server in order to grant the user Golden
	// VCR Fun Points equivalent to the number of bits they cheered with
	_, err = h.ledgerClient.RequestCreditFromCheer(ctx, accessToken, ev.Bits, ev.Message)
	if err != nil {
		return fmt.Errorf("RequestCreditFromCheer failed: %v", err)
	}

	return nil
}
