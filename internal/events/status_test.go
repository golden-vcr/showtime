package events

import (
	"errors"
	"net/http"
	"testing"

	"github.com/golden-vcr/showtime/internal/twitch"
	"github.com/nicklaw5/helix/v2"
	"github.com/stretchr/testify/assert"
)

func Test_VerifySubscriptionStatus(t *testing.T) {
	channelUserId := "1337"
	webhookCallbackUrl := "https://example.com/webhook"
	tests := []struct {
		name                   string
		c                      twitch.SubscriptionReader
		required               []RequiredSubscription
		wantErr                error
		wantSecondaryErrSubstr string
	}{
		{
			"returns ErrFailedToGetSubscriptions if Twitch API call fails",
			&mockSubscriptionReader{
				err: errors.New("mock Twitch API error"),
			},
			nil,
			ErrFailedToGetSubscriptions,
			"mock Twitch API error",
		},
		{
			"returns ErrNoSubscriptionsExist if Twitch API returns no subscriptions",
			&mockSubscriptionReader{},
			nil,
			ErrNoSubscriptionsExist,
			"",
		},
		{
			"returns ErrMissingSubscriptions if any required subscriptions do not exist",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID:      "foo",
						Type:    helix.EventSubTypeChannelFollow,
						Version: "2",
						Status:  helix.EventSubStatusEnabled,
						Condition: helix.EventSubCondition{
							BroadcasterUserID: "1337",
							ModeratorUserID:   "1337",
						},
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
				},
			},
			[]RequiredSubscription{
				{
					Type:    helix.EventSubTypeChannelUpdate,
					Version: "2",
					TemplatedCondition: helix.EventSubCondition{
						BroadcasterUserID: "{{.ChannelUserId}}",
					},
				},
			},
			ErrMissingSubscriptions,
			"",
		},
		{
			"returns ErrSubscriptionsDisabled if any required subscriptions do not show enabled status",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID:      "foo",
						Type:    helix.EventSubTypeChannelFollow,
						Version: "2",
						Status:  helix.EventSubStatusAuthorizationRevoked,
						Condition: helix.EventSubCondition{
							BroadcasterUserID: "1337",
							ModeratorUserID:   "1337",
						},
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
				},
			},
			[]RequiredSubscription{
				{
					Type:    helix.EventSubTypeChannelFollow,
					Version: "2",
					TemplatedCondition: helix.EventSubCondition{
						BroadcasterUserID: "{{.ChannelUserId}}",
						ModeratorUserID:   "{{.ChannelUserId}}",
					},
				},
			},
			ErrSubscriptionsDisabled,
			"",
		},
		{
			"completes without error if all required subscriptions exist and are enabled",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID:      "foo",
						Type:    helix.EventSubTypeChannelFollow,
						Version: "2",
						Status:  helix.EventSubStatusEnabled,
						Condition: helix.EventSubCondition{
							BroadcasterUserID: "1337",
							ModeratorUserID:   "1337",
						},
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
				},
			},
			[]RequiredSubscription{
				{
					Type:    helix.EventSubTypeChannelFollow,
					Version: "2",
					TemplatedCondition: helix.EventSubCondition{
						BroadcasterUserID: "{{.ChannelUserId}}",
						ModeratorUserID:   "{{.ChannelUserId}}",
					},
				},
			},
			nil,
			"",
		},
		{
			"ignores subscriptions that do not match the required condition",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID:      "foo",
						Type:    helix.EventSubTypeChannelFollow,
						Version: "2",
						Status:  helix.EventSubStatusEnabled,
						Condition: helix.EventSubCondition{
							BroadcasterUserID: "10000",
							ModeratorUserID:   "10000",
						},
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
				},
			},
			[]RequiredSubscription{
				{
					Type:    helix.EventSubTypeChannelFollow,
					Version: "2",
					TemplatedCondition: helix.EventSubCondition{
						BroadcasterUserID: "{{.ChannelUserId}}",
						ModeratorUserID:   "{{.ChannelUserId}}",
					},
				},
			},
			ErrMissingSubscriptions,
			"",
		},
		{
			"ignores subscriptions that do not match the configured callback URL",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID:      "foo",
						Type:    helix.EventSubTypeChannelFollow,
						Version: "2",
						Status:  helix.EventSubStatusEnabled,
						Condition: helix.EventSubCondition{
							BroadcasterUserID: "1337",
							ModeratorUserID:   "1337",
						},
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://an-entirely-different-domain.com/webhook",
						},
					},
				},
			},
			[]RequiredSubscription{
				{
					Type:    helix.EventSubTypeChannelFollow,
					Version: "2",
					TemplatedCondition: helix.EventSubCondition{
						BroadcasterUserID: "{{.ChannelUserId}}",
						ModeratorUserID:   "{{.ChannelUserId}}",
					},
				},
			},
			ErrNoSubscriptionsExist,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err, secondaryErr := VerifySubscriptionStatus(tt.c, tt.required, channelUserId, webhookCallbackUrl)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				if tt.wantSecondaryErrSubstr != "" {
					assert.ErrorContains(t, secondaryErr, tt.wantSecondaryErrSubstr)
				} else {
					assert.NoError(t, secondaryErr)
				}
			} else {
				assert.NoError(t, err)
				assert.NoError(t, secondaryErr)
			}
		})
	}
}

func Test_GetOwnedSubscriptions(t *testing.T) {
	channelUserId := "1337"
	webhookCallbackUrl := "https://example.com/webhook"
	tests := []struct {
		name         string
		c            twitch.SubscriptionReader
		wantErr      bool
		wantMatchIds []string
	}{
		{
			"abort and returns error if Twitch API call fails",
			&mockSubscriptionReader{
				err: errors.New("mock Twitch API error"),
			},
			true,
			nil,
		},
		{
			"aborts and returns error if Twitch API gives non-200 response",
			&mockSubscriptionReader{
				statusCode: http.StatusBadRequest,
			},
			true,
			nil,
		},
		{
			"accepts all relevant subscriptions returned from Twitch API",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID: "foo",
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
					{
						ID: "bar",
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
				},
			},
			false,
			[]string{"foo", "bar"},
		},
		{
			"ignores subscriptions that don't match webhook callback",
			&mockSubscriptionReader{
				subscriptions: []helix.EventSubSubscription{
					{
						ID: "foo",
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://example.com/webhook",
						},
					},
					{
						ID: "bar",
						Transport: helix.EventSubTransport{
							Method:   "webhook",
							Callback: "https://an-entirely-different-domain.com/webhook",
						},
					},
				},
			},
			false,
			[]string{"foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owned, err := GetOwnedSubscriptions(tt.c, channelUserId, webhookCallbackUrl)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				matchIds := make([]string, 0, len(owned))
				for _, subscription := range owned {
					matchIds = append(matchIds, subscription.ID)
				}
				assert.ElementsMatch(t, matchIds, tt.wantMatchIds)
			}
		})
	}
}

func Test_ReconcileRequiredSubscriptions(t *testing.T) {
	required := []RequiredSubscription{
		{
			Type:    helix.EventSubTypeChannelUpdate,
			Version: "2",
			TemplatedCondition: helix.EventSubCondition{
				BroadcasterUserID: "{{.ChannelUserId}}",
			},
		},
		{
			Type:    helix.EventSubTypeChannelRaid,
			Version: "1",
			TemplatedCondition: helix.EventSubCondition{
				ToBroadcasterUserID: "{{.ChannelUserId}}",
			},
		},
	}
	owned := []helix.EventSubSubscription{
		{
			ID:      "foo",
			Type:    helix.EventSubTypeChannelUpdate,
			Version: "2",
			Condition: helix.EventSubCondition{
				BroadcasterUserID: "1337",
			},
		},
		{
			ID:      "bar",
			Type:    helix.EventSubTypeChannelFollow,
			Version: "2",
			Condition: helix.EventSubCondition{
				BroadcasterUserID: "1337",
				ModeratorUserID:   "1337",
			},
		},
	}
	reconciled, err := ReconcileRequiredSubscriptions(required, owned, "1337")
	assert.NoError(t, err)
	assert.NotNil(t, reconciled)
	assert.Len(t, reconciled.Existing, 1)
	assert.Equal(t, "foo", reconciled.Existing[0].Value.ID)
	assert.Equal(t, helix.EventSubTypeChannelUpdate, reconciled.Existing[0].Value.Type)
	assert.Equal(t, helix.EventSubTypeChannelUpdate, reconciled.Existing[0].Required.Type)
	assert.Len(t, reconciled.ToCreate, 1)
	assert.Equal(t, helix.EventSubTypeChannelRaid, reconciled.ToCreate[0].Type)
	assert.Len(t, reconciled.ToDelete, 1)
	assert.Equal(t, "bar", reconciled.ToDelete[0].ID)
	assert.Equal(t, helix.EventSubTypeChannelFollow, reconciled.ToDelete[0].Type)

	reconciledFromEmpty, err := ReconcileRequiredSubscriptions(required, nil, "1337")
	assert.NoError(t, err)
	assert.NotNil(t, reconciled)
	assert.Len(t, reconciledFromEmpty.Existing, 0)
	assert.Len(t, reconciledFromEmpty.ToCreate, 2)
	assert.Len(t, reconciledFromEmpty.ToDelete, 0)
}

func Test_findMatchingSubscription(t *testing.T) {
	requiredType := helix.EventSubTypeChannelFollow
	requiredVersion := "2"
	requiredCondition := &helix.EventSubCondition{
		BroadcasterUserID: "1337",
		ModeratorUserID:   "1337",
	}
	tests := []struct {
		name          string
		subscriptions []helix.EventSubSubscription
		wantMatchId   string
	}{
		{
			"no match when empty",
			[]helix.EventSubSubscription{},
			"",
		},
		{
			"exact match",
			[]helix.EventSubSubscription{
				{
					ID:      "foo",
					Type:    helix.EventSubTypeChannelFollow,
					Version: "2",
					Condition: helix.EventSubCondition{
						BroadcasterUserID: "1337",
						ModeratorUserID:   "1337",
					},
				},
			},
			"foo",
		},
		{
			"no match if version differs",
			[]helix.EventSubSubscription{
				{
					ID:      "foo",
					Type:    helix.EventSubTypeChannelFollow,
					Version: "1",
					Condition: helix.EventSubCondition{
						BroadcasterUserID: "1337",
						ModeratorUserID:   "1337",
					},
				},
			},
			"",
		},
		{
			"no match if type differs",
			[]helix.EventSubSubscription{
				{
					ID:      "foo",
					Type:    helix.EventSubTypeChannelSubscription,
					Version: "2",
					Condition: helix.EventSubCondition{
						BroadcasterUserID: "1337",
						ModeratorUserID:   "1337",
					},
				},
			},
			"",
		},
		{
			"no match if condition differs",
			[]helix.EventSubSubscription{
				{
					ID:      "foo",
					Type:    helix.EventSubTypeChannelFollow,
					Version: "2",
					Condition: helix.EventSubCondition{
						BroadcasterUserID: "1337",
						ModeratorUserID:   "1300",
					},
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMatch := findMatchingSubscription(tt.subscriptions, requiredType, requiredVersion, requiredCondition)
			if tt.wantMatchId == "" {
				assert.Nil(t, gotMatch)
			} else {
				assert.NotNil(t, gotMatch)
				assert.Equal(t, requiredType, gotMatch.Type)
				assert.Equal(t, requiredVersion, gotMatch.Version)
				assert.Equal(t, tt.wantMatchId, gotMatch.ID)
			}
		})
	}
}

type mockSubscriptionReader struct {
	subscriptions []helix.EventSubSubscription
	statusCode    int
	err           error
}

func (m *mockSubscriptionReader) GetEventSubSubscriptions(params *helix.EventSubSubscriptionsParams) (*helix.EventSubSubscriptionsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	statusCode := m.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	return &helix.EventSubSubscriptionsResponse{
		ResponseCommon: helix.ResponseCommon{StatusCode: statusCode},
		Data: helix.ManyEventSubSubscriptions{
			Total:                 len(m.subscriptions),
			EventSubSubscriptions: m.subscriptions,
		},
	}, nil
}
