// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: screening.sql

package queries

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const getCurrentScreening = `-- name: GetCurrentScreening :one
select
    screening.id::uuid as id,
    screening.tape_id,
    screening.started_at,
    screening.ended_at
from showtime.broadcast
join showtime.screening
    on screening.broadcast_id = broadcast.id
order by screening.started_at desc, broadcast.started_at desc
limit 1
`

type GetCurrentScreeningRow struct {
	ID        uuid.UUID
	TapeID    int32
	StartedAt time.Time
	EndedAt   sql.NullTime
}

func (q *Queries) GetCurrentScreening(ctx context.Context) (GetCurrentScreeningRow, error) {
	row := q.db.QueryRowContext(ctx, getCurrentScreening)
	var i GetCurrentScreeningRow
	err := row.Scan(
		&i.ID,
		&i.TapeID,
		&i.StartedAt,
		&i.EndedAt,
	)
	return i, err
}

const getMostRecentScreening = `-- name: GetMostRecentScreening :one
select
    screening.tape_id,
    screening.started_at,
    screening.ended_at
from showtime.screening
where screening.broadcast_id = $1
order by screening.started_at desc
limit 1
`

type GetMostRecentScreeningRow struct {
	TapeID    int32
	StartedAt time.Time
	EndedAt   sql.NullTime
}

func (q *Queries) GetMostRecentScreening(ctx context.Context, broadcastID int32) (GetMostRecentScreeningRow, error) {
	row := q.db.QueryRowContext(ctx, getMostRecentScreening, broadcastID)
	var i GetMostRecentScreeningRow
	err := row.Scan(&i.TapeID, &i.StartedAt, &i.EndedAt)
	return i, err
}

const recordScreeningEnded = `-- name: RecordScreeningEnded :exec
update showtime.screening set ended_at = now()
where screening.broadcast_id = $1
    and screening.ended_at is null
`

func (q *Queries) RecordScreeningEnded(ctx context.Context, broadcastID int32) error {
	_, err := q.db.ExecContext(ctx, recordScreeningEnded, broadcastID)
	return err
}

const recordScreeningStarted = `-- name: RecordScreeningStarted :exec
insert into showtime.screening (
    broadcast_id,
    tape_id,
    started_at
) values (
    $1,
    $2,
    now()
)
`

type RecordScreeningStartedParams struct {
	BroadcastID int32
	TapeID      int32
}

func (q *Queries) RecordScreeningStarted(ctx context.Context, arg RecordScreeningStartedParams) error {
	_, err := q.db.ExecContext(ctx, recordScreeningStarted, arg.BroadcastID, arg.TapeID)
	return err
}
