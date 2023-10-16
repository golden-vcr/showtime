-- name: RecordScreeningStarted :exec
insert into showtime.screening (
    broadcast_id,
    tape_id,
    started_at
) values (
    sqlc.arg('broadcast_id'),
    sqlc.arg('tape_id'),
    now()
);

-- name: RecordScreeningEnded :exec
update showtime.screening set ended_at = now()
where screening.broadcast_id = sqlc.arg('broadcast_id')
    and screening.ended_at is null;

-- name: GetCurrentTapeId :one
select
    (case when screening.ended_at is null
        then screening.tape_id
        else null
    end)::integer as tape_id
from showtime.screening
where screening.broadcast_id = (
    select broadcast.id from showtime.broadcast
    order by broadcast.started_at desc limit 1
)
order by screening.started_at desc limit 1;
