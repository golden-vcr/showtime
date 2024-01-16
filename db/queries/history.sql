-- name: GetBroadcastHistory :many
select
    broadcast.id,
    broadcast.started_at,
    broadcast.vod_url,
    array_agg(screening.tape_id order by screening.started_at)::integer[] as tape_ids
from showtime.broadcast
join showtime.screening
    on screening.broadcast_id = broadcast.id
group by broadcast.id
order by broadcast.started_at;

-- name: GetTapeScreeningHistory :many
select
    screening.tape_id,
    array_agg(
        distinct screening.broadcast_id
        order by screening.broadcast_id
    )::integer[] as broadcast_ids
from showtime.screening
group by screening.tape_id
order by screening.tape_id;

-- name: GetBroadcastById :one
select
    *
from showtime.broadcast
where broadcast.id = sqlc.arg('broadcast_id');

-- name: GetScreeningsByBroadcastId :many
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
where screening.broadcast_id = sqlc.arg('broadcast_id')
order by screening.started_at;

-- name: GetViewerLookupForBroadcast :many
select
    viewer.twitch_user_id,
    viewer.twitch_display_name
from showtime.viewer
where viewer.twitch_user_id in (
    select distinct image_request.twitch_user_id
    from showtime.image_request
    where image_request.screening_id in (
        select screening.id from showtime.screening
        where screening.broadcast_id = sqlc.arg('broadcast_id')
    )
);
