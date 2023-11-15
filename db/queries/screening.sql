-- name: GetMostRecentScreening :one
select
    screening.tape_id,
    screening.started_at,
    screening.ended_at
from showtime.screening
where screening.broadcast_id = sqlc.arg('broadcast_id')
order by screening.started_at desc
limit 1;

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

-- name: GetCurrentScreening :one
select
    screening.id::uuid as id,
    screening.tape_id,
    screening.started_at,
    screening.ended_at
from showtime.broadcast
join showtime.screening
    on screening.broadcast_id = broadcast.id
order by screening.started_at desc, broadcast.started_at desc
limit 1;
