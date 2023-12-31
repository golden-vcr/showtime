// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: history.sql

package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

const getBroadcastById = `-- name: GetBroadcastById :one
select
    id, started_at, ended_at, vod_url
from showtime.broadcast
where broadcast.id = $1
`

func (q *Queries) GetBroadcastById(ctx context.Context, broadcastID int32) (ShowtimeBroadcast, error) {
	row := q.db.QueryRowContext(ctx, getBroadcastById, broadcastID)
	var i ShowtimeBroadcast
	err := row.Scan(
		&i.ID,
		&i.StartedAt,
		&i.EndedAt,
		&i.VodUrl,
	)
	return i, err
}

const getScreeningsByBroadcastId = `-- name: GetScreeningsByBroadcastId :many
select
    screening.tape_id,
    screening.started_at,
    screening.ended_at,
    coalesce(
        (
            select json_agg(
                json_build_object(
                    'id', image_request.id,
                    'twitch_user_id', twitch_user_id,
                    'subject', subject_noun_clause
                ))
            from showtime.image_request
            where image_request.screening_id = screening.id
                and image_request.finished_at is not null
                and image_request.error_message is null
        ),
        '[]'::json
    )::json as image_requests
from showtime.screening
where screening.broadcast_id = $1
order by screening.started_at
`

type GetScreeningsByBroadcastIdRow struct {
	TapeID        int32
	StartedAt     time.Time
	EndedAt       sql.NullTime
	ImageRequests json.RawMessage
}

func (q *Queries) GetScreeningsByBroadcastId(ctx context.Context, broadcastID int32) ([]GetScreeningsByBroadcastIdRow, error) {
	rows, err := q.db.QueryContext(ctx, getScreeningsByBroadcastId, broadcastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetScreeningsByBroadcastIdRow
	for rows.Next() {
		var i GetScreeningsByBroadcastIdRow
		if err := rows.Scan(
			&i.TapeID,
			&i.StartedAt,
			&i.EndedAt,
			&i.ImageRequests,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTapeScreeningHistory = `-- name: GetTapeScreeningHistory :many
select
    screening.tape_id,
    array_agg(
        distinct screening.broadcast_id
        order by screening.broadcast_id
    )::integer[] as broadcast_ids
from showtime.screening
group by screening.tape_id
order by screening.tape_id
`

type GetTapeScreeningHistoryRow struct {
	TapeID       int32
	BroadcastIds []int32
}

func (q *Queries) GetTapeScreeningHistory(ctx context.Context) ([]GetTapeScreeningHistoryRow, error) {
	rows, err := q.db.QueryContext(ctx, getTapeScreeningHistory)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetTapeScreeningHistoryRow
	for rows.Next() {
		var i GetTapeScreeningHistoryRow
		if err := rows.Scan(&i.TapeID, pq.Array(&i.BroadcastIds)); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getViewerLookupForBroadcast = `-- name: GetViewerLookupForBroadcast :many
select
    viewer.twitch_user_id,
    viewer.twitch_display_name
from showtime.viewer
where viewer.twitch_user_id in (
    select distinct image_request.twitch_user_id
    from showtime.image_request
    where image_request.screening_id in (
        select screening.id from showtime.screening
        where screening.broadcast_id = $1
    )
)
`

type GetViewerLookupForBroadcastRow struct {
	TwitchUserID      string
	TwitchDisplayName string
}

func (q *Queries) GetViewerLookupForBroadcast(ctx context.Context, broadcastID int32) ([]GetViewerLookupForBroadcastRow, error) {
	rows, err := q.db.QueryContext(ctx, getViewerLookupForBroadcast, broadcastID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetViewerLookupForBroadcastRow
	for rows.Next() {
		var i GetViewerLookupForBroadcastRow
		if err := rows.Scan(&i.TwitchUserID, &i.TwitchDisplayName); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
