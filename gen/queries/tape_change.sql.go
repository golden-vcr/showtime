// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: tape_change.sql

package queries

import (
	"context"
)

const clearTapeId = `-- name: ClearTapeId :exec
insert into showtime.tape_change (tape_id) values ('')
`

func (q *Queries) ClearTapeId(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, clearTapeId)
	return err
}

const getCurrentTapeId = `-- name: GetCurrentTapeId :one
select tape_change.tape_id
from showtime.tape_change
order by tape_change.created_at desc
limit 1
`

func (q *Queries) GetCurrentTapeId(ctx context.Context) (string, error) {
	row := q.db.QueryRowContext(ctx, getCurrentTapeId)
	var tape_id string
	err := row.Scan(&tape_id)
	return tape_id, err
}

const setTapeId = `-- name: SetTapeId :exec
insert into showtime.tape_change (tape_id) values ($1::text)
`

func (q *Queries) SetTapeId(ctx context.Context, tapeID string) error {
	_, err := q.db.ExecContext(ctx, setTapeId, tapeID)
	return err
}