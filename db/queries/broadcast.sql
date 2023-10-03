-- name: GetMostRecentBroadcast :one
select
    broadcast.id,
    broadcast.started_at,
    broadcast.ended_at
from showtime.broadcast
order by broadcast.started_at desc
limit 1;

-- name: RecordBroadcastStarted :one
insert into showtime.broadcast (
    started_at
) values (
    now()
)
returning broadcast.id;

-- name: RecordBroadcastResumed :exec
update showtime.broadcast set ended_at = null
where broadcast.id = sqlc.arg('broadcast_id')
    and broadcast.ended_at is not null;

-- name: RecordBroadcastEnded :exec
with most_recent_broadcast as (
    select broadcast.id from showtime.broadcast
    order by broadcast.started_at desc
    limit 1
)
update showtime.broadcast set ended_at = now()
where broadcast.id = (select id from most_recent_broadcast)
    and broadcast.ended_at is null;
