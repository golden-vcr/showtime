package queries_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golden-vcr/server-common/querytest"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/stretchr/testify/assert"
)

func Test_GetMostRecentScreening(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.broadcast")
	querytest.AssertCount(t, tx, 0, "SELECT COUNT(*) FROM showtime.screening")

	var broadcastId int32
	rows, err := tx.Query("INSERT INTO showtime.broadcast (started_at) VALUES (now() - '1h'::interval) RETURNING id")
	assert.NoError(t, err)
	assert.True(t, rows.Next())
	err = rows.Scan(&broadcastId)
	assert.NoError(t, err)
	assert.Greater(t, broadcastId, int32(0))
	assert.False(t, rows.Next())

	_, err = q.GetMostRecentScreening(context.Background(), broadcastId)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	_, err = tx.Exec(`
		INSERT INTO showtime.screening (broadcast_id, tape_id, started_at, ended_at)
		VALUES ($1, 98, now() - '55m'::interval, now() - '35m'::interval)
	`, broadcastId)
	assert.NoError(t, err)

	screening, err := q.GetMostRecentScreening(context.Background(), broadcastId)
	assert.NoError(t, err)
	assert.Equal(t, int32(98), screening.TapeID)
	assert.True(t, screening.EndedAt.Valid)

	_, err = tx.Exec(`
		INSERT INTO showtime.screening (broadcast_id, tape_id, started_at)
		VALUES ($1, 99, now() - '30m'::interval)
	`, broadcastId)
	assert.NoError(t, err)

	screening, err = q.GetMostRecentScreening(context.Background(), broadcastId)
	assert.NoError(t, err)
	assert.Equal(t, int32(99), screening.TapeID)
	assert.False(t, screening.EndedAt.Valid)
}

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
