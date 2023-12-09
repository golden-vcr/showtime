package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/golden-vcr/auth"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/golden-vcr/showtime/internal/alerts"
	"github.com/golden-vcr/showtime/internal/imagegen"
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

	fmt.Printf("Generating alert for follow from user %s\n", ev.UserName)
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

	fmt.Printf("Generating alert for raid by broadcaster %s with %d viewers\n", ev.FromBroadcasterUserName, ev.Viewers)
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
	fmt.Printf("Requesting JWT in response to cheer by user %s\n", ev.UserName)
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
	fmt.Printf("Requesting credit of %d points to user %s\n", ev.Bits, ev.UserName)
	_, err = h.ledgerClient.RequestCreditFromCheer(ctx, accessToken, ev.Bits, ev.Message)
	if err != nil {
		return fmt.Errorf("RequestCreditFromCheer failed: %v", err)
	}

	// If the viewer requested that their points be immediately spent on a ghost alert
	// (by cheering with 200 bits and including "ghost of <subject>" in the message),
	// submit a request to the imagegen server on their behalf
	ghostPrefix := "ghost of "
	ghostSubject := ""
	if ev.Bits == 200 {
		ghostPrefixPos := strings.Index(ghostPrefix, ghostPrefix)
		if ghostPrefixPos >= 0 {
			ghostSubject = strings.TrimSpace(ev.Message[ghostPrefixPos+len(ghostPrefix):])
		}
	}
	if ghostSubject != "" {
		fmt.Printf("Requesting ghost alert on behalf of user %s, with subject '%s'\n", ev.UserName, ghostSubject)
		go func() {
			payload := imagegen.Request{Subject: ghostSubject}
			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				fmt.Printf("Failed to marshal request payload when processing ghost alert for %s: %v\n", ev.UserName, err)
				return
			}
			req, err := http.NewRequestWithContext(h.imagegenCtx, http.MethodPost, h.imagegenUrl, bytes.NewReader(payloadBytes))
			if err != nil {
				fmt.Printf("Failed to initialize HTTP request when processing ghost alert for %s: %v\n", ev.UserName, err)
				return
			}
			req.Header.Set("authorization", fmt.Sprintf("Bearer %s", accessToken))

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Failed to complete HTTP request when processing ghost alert for %s: %v\n", ev.UserName, err)
				return
			}
			if res.StatusCode < 200 || res.StatusCode > 299 {
				suffix := ""
				if body, err := io.ReadAll(res.Body); err == nil {
					suffix = fmt.Sprintf(": %s", body)
				}
				fmt.Printf("Ghost alert for %s failed with status %d%s\n", ev.UserName, res.StatusCode, suffix)
				return
			}
			fmt.Printf("Successfully submitted ghost alert on behalf of %s\n", ev.UserName)
		}()
	}
	return nil
}
