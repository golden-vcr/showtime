-- name: RecordViewerLogin :exec
insert into showtime.viewer (
    twitch_user_id,
    twitch_display_name,
    first_logged_in_at,
    last_logged_in_at
) values (
    sqlc.arg('twitch_user_id'),
    sqlc.arg('twitch_display_name'),
    now(),
    now()
)
on conflict (twitch_user_id) do update set
    twitch_display_name = excluded.twitch_display_name,
    last_logged_in_at = excluded.last_logged_in_at;

-- name: RecordViewerFollow :exec
insert into showtime.viewer (
    twitch_user_id,
    twitch_display_name,
    first_followed_at
) values (
    sqlc.arg('twitch_user_id'),
    sqlc.arg('twitch_display_name'),
    now()
)
on conflict (twitch_user_id) do update set
    twitch_display_name = excluded.twitch_display_name,
    first_followed_at = coalesce(viewer.first_followed_at, excluded.first_followed_at);

