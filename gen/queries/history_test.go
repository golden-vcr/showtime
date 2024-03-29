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

func Test_GetTapeBroadcastHistory(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	// We should have no broadcast history initially
	rows, err := q.GetBroadcastHistory(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rows, 0)

	// Simulate three broadcasts
	_, err = tx.Exec(`
		INSERT INTO showtime.broadcast (id, started_at, ended_at, vod_url) VALUES
			(1, now() - '12h'::interval, now() - '10h'::interval, 'https://vods.com/1'),
			(2, now() - '6h'::interval, now() - '4h'::interval, NULL),
			(3, now() - '2h'::interval, NULL, NULL);
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
	rows, err = q.GetBroadcastHistory(context.Background())
	assert.NoError(t, err)
	assert.Len(t, rows, 3)
	assert.Equal(t, int32(1), rows[0].ID)
	assert.Equal(t, int32(2), rows[1].ID)
	assert.Equal(t, int32(3), rows[2].ID)
	assert.Greater(t, rows[1].StartedAt, rows[0].StartedAt)
	assert.Greater(t, rows[2].StartedAt, rows[1].StartedAt)
	assert.True(t, rows[0].VodUrl.Valid)
	assert.Equal(t, "https://vods.com/1", rows[0].VodUrl.String)
	assert.False(t, rows[1].VodUrl.Valid)
	assert.False(t, rows[2].VodUrl.Valid)
	assert.Equal(t, []int32{40, 50}, rows[0].TapeIds)
	assert.Equal(t, []int32{60}, rows[1].TapeIds)
	assert.Equal(t, []int32{40, 70}, rows[2].TapeIds)
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

func Test_GetViewerLookupForBroadcast(t *testing.T) {
	tx := querytest.PrepareTx(t)
	q := queries.New(tx)

	_, err := tx.Exec(`INSERT INTO showtime.broadcast (id, started_at, ended_at) VALUES (
		1,
		now() - '12h'::interval,
		now() - '10h'::interval
	)`)
	assert.NoError(t, err)

	_, err = tx.Exec(`INSERT INTO showtime.screening (id, broadcast_id, tape_id, started_at, ended_at) VALUES (
		'effec22c-c0d1-429b-a91e-9248e069e19a',
		1,
		44,
		now() - '11h30m'::interval,
		now() - '11h'::interval
	)`)
	assert.NoError(t, err)

	_, err = tx.Exec(`
		INSERT INTO showtime.viewer (twitch_user_id, twitch_display_name) VALUES
			('51234', 'triangle_man'),
			('99009', 'UniverseMan')
	`)
	assert.NoError(t, err)

	_, err = tx.Exec(`
		INSERT INTO showtime.image_request (id, twitch_user_id, subject_noun_clause, prompt, screening_id) VALUES
			('e5f4b298-99c8-4f81-aada-1132e74f1d6d', '51234', 'foo', 'foobar', 'effec22c-c0d1-429b-a91e-9248e069e19a'),
			('54d3fdd8-5234-4b80-8acc-6b9dca28aac6', '99009', 'foo', 'foobar', 'effec22c-c0d1-429b-a91e-9248e069e19a')
	`)
	assert.NoError(t, err)

	rows, err := q.GetViewerLookupForBroadcast(context.Background(), 1)
	assert.NoError(t, err)
	assert.ElementsMatch(t, rows, []queries.GetViewerLookupForBroadcastRow{
		{TwitchUserID: "51234", TwitchDisplayName: "triangle_man"},
		{TwitchUserID: "99009", TwitchDisplayName: "UniverseMan"},
	})
}
