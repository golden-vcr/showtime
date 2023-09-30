// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.20.0
// source: viewer.sql

package queries

import (
	"context"
)

const recordViewerFollow = `-- name: RecordViewerFollow :exec
insert into showtime.viewer (
    twitch_user_id,
    twitch_display_name,
    first_followed_at
) values (
    $1,
    $2,
    now()
)
on conflict (twitch_user_id) do update set
    twitch_display_name = excluded.twitch_display_name,
    first_followed_at = coalesce(viewer.first_followed_at, excluded.first_followed_at)
`

type RecordViewerFollowParams struct {
	TwitchUserID      string
	TwitchDisplayName string
}

func (q *Queries) RecordViewerFollow(ctx context.Context, arg RecordViewerFollowParams) error {
	_, err := q.db.ExecContext(ctx, recordViewerFollow, arg.TwitchUserID, arg.TwitchDisplayName)
	return err
}

const recordViewerLogin = `-- name: RecordViewerLogin :exec
insert into showtime.viewer (
    twitch_user_id,
    twitch_display_name,
    first_logged_in_at,
    last_logged_in_at
) values (
    $1,
    $2,
    now(),
    now()
)
on conflict (twitch_user_id) do update set
    twitch_display_name = excluded.twitch_display_name,
    last_logged_in_at = excluded.last_logged_in_at
`

type RecordViewerLoginParams struct {
	TwitchUserID      string
	TwitchDisplayName string
}

func (q *Queries) RecordViewerLogin(ctx context.Context, arg RecordViewerLoginParams) error {
	_, err := q.db.ExecContext(ctx, recordViewerLogin, arg.TwitchUserID, arg.TwitchDisplayName)
	return err
}
