// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: image.sql

package queries

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

const recordImage = `-- name: RecordImage :exec
insert into showtime.image (
    image_request_id,
    index,
    url
) values (
    $1,
    $2,
    $3
)
`

type RecordImageParams struct {
	ImageRequestID uuid.UUID
	Index          int32
	Url            string
}

func (q *Queries) RecordImage(ctx context.Context, arg RecordImageParams) error {
	_, err := q.db.ExecContext(ctx, recordImage, arg.ImageRequestID, arg.Index, arg.Url)
	return err
}

const recordImageRequest = `-- name: RecordImageRequest :exec
insert into showtime.image_request (
    id,
    twitch_user_id,
    subject_noun_clause,
    prompt,
    created_at
) values (
    $1,
    $2,
    $3,
    $4,
    now()
)
`

type RecordImageRequestParams struct {
	ImageRequestID    uuid.UUID
	TwitchUserID      string
	SubjectNounClause string
	Prompt            string
}

func (q *Queries) RecordImageRequest(ctx context.Context, arg RecordImageRequestParams) error {
	_, err := q.db.ExecContext(ctx, recordImageRequest,
		arg.ImageRequestID,
		arg.TwitchUserID,
		arg.SubjectNounClause,
		arg.Prompt,
	)
	return err
}

const recordImageRequestFailure = `-- name: RecordImageRequestFailure :execresult
update showtime.image_request set
    finished_at = now(),
    error_message = $1::text
where image_request.id = $2
    and finished_at is null
`

type RecordImageRequestFailureParams struct {
	ErrorMessage   string
	ImageRequestID uuid.UUID
}

func (q *Queries) RecordImageRequestFailure(ctx context.Context, arg RecordImageRequestFailureParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, recordImageRequestFailure, arg.ErrorMessage, arg.ImageRequestID)
}

const recordImageRequestSuccess = `-- name: RecordImageRequestSuccess :execresult
update showtime.image_request set
    finished_at = now()
where image_request.id = $1
    and finished_at is null
`

func (q *Queries) RecordImageRequestSuccess(ctx context.Context, imageRequestID uuid.UUID) (sql.Result, error) {
	return q.db.ExecContext(ctx, recordImageRequestSuccess, imageRequestID)
}
