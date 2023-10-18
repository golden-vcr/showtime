package queries_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golden-vcr/server-common/querytest"
	"github.com/golden-vcr/showtime/gen/queries"
	"github.com/stretchr/testify/assert"
)

func Test_GetTapeScreeningHistory(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	// We should have no screening history initially
	rows, err := q.GetTapeScreeningHistory(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rows, 0)

	// Simulate three broadcasts
	_, err = tx.Exec(`
		INSERT INTO showtime.broadcast (id, started_at, ended_at) VALUES
			(1, now() - '12h'::interval, now() - '10h'::interval),
			(2, now() - '6h'::interval, now() - '4h'::interval),
			(3, now() - '2h'::interval, NULL);
	`)
	assert.NoError(t, err)

	// Simulate screenings within those broadcasts: tapes 40 and 50 in broadcast 1,
	// then tape 60 in broadcast 2, then 70 and a repeat of 40 in broadcast 3 (which is
	// ongoing)
	_, err = tx.Exec(`
		INSERT INTO showtime.screening (broadcast_id, tape_id, started_at, ended_at) VALUES
			(1, 40, now() - '11h30m'::interval, now() - '11h'::interval),
			(1, 50, now() - '11h'::interval, now() - '10h30m'::interval),
			(2, 60, now() - '6h'::interval, now() - '5h'::interval),
			(3, 40, now() - '2h'::interval, now() - '1h'::interval),
			(3, 70, now() - '30m'::interval, NULL);
	`)
	assert.NoError(t, err)

	// Our screening history should now reflect our state, with entries for the 4 unique
	// tapes that we've screened
	rows, err = q.GetTapeScreeningHistory(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []queries.GetTapeScreeningHistoryRow{
		{
			TapeID:       40,
			BroadcastIds: []int32{1, 3},
		},
		{
			TapeID:       50,
			BroadcastIds: []int32{1},
		},
		{
			TapeID:       60,
			BroadcastIds: []int32{2},
		},
		{
			TapeID:       70,
			BroadcastIds: []int32{3},
		},
	}, rows)
}

func Test_GetBroadcastById(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	_, err := q.GetBroadcastById(context.Background(), 1)
	assert.ErrorIs(t, err, sql.ErrNoRows)

	_, err = tx.Exec(`
		INSERT INTO showtime.broadcast (id, started_at, ended_at) VALUES
			(1, now() - '12h'::interval, now() - '10h'::interval)
	`)
	assert.NoError(t, err)

	broadcast, err := q.GetBroadcastById(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), broadcast.ID)
	assert.True(t, broadcast.EndedAt.Valid)
	assert.Equal(t, 2*time.Hour, broadcast.EndedAt.Time.Sub(broadcast.StartedAt))
}

func Test_GetScreeningsByBroadcastId(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	screenings, err := q.GetScreeningsByBroadcastId(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, screenings, 0)

	_, err = tx.Exec(`
		INSERT INTO showtime.broadcast (id, started_at, ended_at) VALUES
			(1, now() - '12h'::interval, now() - '10h'::interval)
	`)
	assert.NoError(t, err)
	_, err = tx.Exec(`
		INSERT INTO showtime.screening (broadcast_id, tape_id, started_at, ended_at) VALUES
			(1, 99, now() - '11h45m'::interval, now() - '11h'::interval),
			(1, 15, now() - '11h'::interval, now() - '10h30m'::interval);
	`)
	assert.NoError(t, err)

	screenings, err = q.GetScreeningsByBroadcastId(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, screenings, 2)
	assert.Equal(t, int32(99), screenings[0].TapeID)
	assert.Equal(t, int32(15), screenings[1].TapeID)
	assert.True(t, screenings[0].EndedAt.Valid)
	assert.True(t, screenings[1].EndedAt.Valid)
	assert.Equal(t, 45*time.Minute, screenings[0].EndedAt.Time.Sub(screenings[0].StartedAt))
	assert.Equal(t, 30*time.Minute, screenings[1].EndedAt.Time.Sub(screenings[1].StartedAt))
}
