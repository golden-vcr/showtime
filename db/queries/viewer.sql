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

