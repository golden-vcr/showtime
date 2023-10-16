package queries_test

import (
	"context"
	"testing"

	"github.com/golden-vcr/server-common/querytest"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/stretchr/testify/assert"
)

func Test_RecordScreeningStarted(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.broadcast")
	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.screening")

	broadcastId, err := q.RecordBroadcastStarted(context.Background())
	assert.NoError(t, err)

	err = q.RecordScreeningStarted(context.Background(), queries.RecordScreeningStartedParams{
		BroadcastID: broadcastId,
		TapeID:      42,
	})
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.screening
			WHERE broadcast_id = $1
			AND tape_id = 42
			AND ended_at IS NULL
	`, broadcastId)
}

func Test_RecordScreeningEnded(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.screening")

	broadcastId, err := q.RecordBroadcastStarted(context.Background())
	assert.NoError(t, err)

	err = q.RecordScreeningStarted(context.Background(), queries.RecordScreeningStartedParams{
		BroadcastID: broadcastId,
		TapeID:      101,
	})
	assert.NoError(t, err)

	err = q.RecordScreeningStarted(context.Background(), queries.RecordScreeningStartedParams{
		BroadcastID: broadcastId,
		TapeID:      102,
	})
	assert.NoError(t, err)

	err = q.RecordScreeningEnded(context.Background(), broadcastId)
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 2, `
		SELECT COUNT(*) FROM showtime.screening
			WHERE broadcast_id = $1
			AND tape_id IN (101, 102)
			AND ended_at IS NOT NULL
	`, broadcastId)
}
