-- name: SetTapeId :exec
insert into showtime.tape_change (tape_id) values (@tape_id::text);

-- name: ClearTapeId :exec
insert into showtime.tape_change (tape_id) values ('');

-- name: GetCurrentTapeId :one
select tape_change.tape_id
from showtime.tape_change
order by tape_change.created_at desc
limit 1;
