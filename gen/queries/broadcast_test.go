package queries_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golden-vcr/server-common/querytest"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/stretchr/testify/assert"
)

func Test_GetMostRecentBroadcast(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	_, err := q.GetMostRecentBroadcast(context.Background())
	assert.ErrorIs(t, err, sql.ErrNoRows)

	_, err = tx.Exec(`
		INSERT INTO showtime.broadcast (id, started_at, ended_at) VALUES (1, now() - '1h'::interval, now())
	`)
	assert.NoError(t, err)

	broadcast, err := q.GetMostRecentBroadcast(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int32(1), broadcast.ID)
	assert.True(t, broadcast.EndedAt.Valid)
}

func Test_RecordBroadcastStarted(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.broadcast")

	broadcastId, err := q.RecordBroadcastStarted(context.Background())
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.broadcast
			WHERE id = $1
			AND ended_at IS NULL
	`, broadcastId)
}

func Test_RecordBroadcastEnded(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.broadcast")

	broadcastId, err := q.RecordBroadcastStarted(context.Background())
	assert.NoError(t, err)
	err = q.RecordBroadcastEnded(context.Background())
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.broadcast
			WHERE id = $1
			AND ended_at IS NOT NULL
	`, broadcastId)
}

func Test_RecordBroadcastResumed(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.broadcast")

	broadcastId, err := q.RecordBroadcastStarted(context.Background())
	assert.NoError(t, err)
	err = q.RecordBroadcastEnded(context.Background())
	assert.NoError(t, err)
	err = q.RecordBroadcastResumed(context.Background(), broadcastId)
	assert.NoError(t, err)

	querytest.AssertCount(t, tx, 1, `
		SELECT COUNT(*) FROM showtime.broadcast
			WHERE id = $1
			AND ended_at IS NULL
	`, broadcastId)
}
