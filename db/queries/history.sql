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
    *
from showtime.screening
where screening.broadcast_id = sqlc.arg('broadcast_id')
order by screening.started_at;
