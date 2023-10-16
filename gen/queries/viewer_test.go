package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/server-common/querytest"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/stretchr/testify/assert"
)

func Test_RecordViewerFollow(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	// We should start with no viewer records
	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.viewer")

	// Recording a new user login should record a new identity, and the first/last login
	// timestamps should be identical initially
	err := q.RecordViewerFollow(context.Background(), queries.RecordViewerFollowParams{
		TwitchUserID:      "1234",
		TwitchDisplayName: "bungus",
	})
	assert.NoError(t, err)
	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.viewer
			WHERE twitch_user_id = '1234' AND twitch_display_name = 'bungus'
			AND first_followed_at = now()
	`)

	// A subsequent login by the same user with a different display name should update
	// the display name
	err = q.RecordViewerFollow(context.Background(), queries.RecordViewerFollowParams{
		TwitchUserID:      "1234",
		TwitchDisplayName: "BunGus",
	})
	assert.NoError(t, err)
	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.viewer WHERE
			twitch_user_id = '1234' AND twitch_display_name = 'BunGus'
	`)

	// We should end up with 1 viewer record
	querytest.AssertCount(t, tx, 1, "SELECT COUNT(*) FROM showtime.viewer")
}
