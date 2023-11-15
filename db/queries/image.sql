-- name: RecordImageRequest :exec
insert into showtime.image_request (
    id,
    twitch_user_id,
    screening_id,
    subject_noun_clause,
    prompt,
    created_at
) values (
    sqlc.arg('image_request_id'),
    sqlc.arg('twitch_user_id'),
    sqlc.narg('screening_id'),
    sqlc.arg('subject_noun_clause'),
    sqlc.arg('prompt'),
    now()
);

-- name: RecordImageRequestFailure :execresult
update showtime.image_request set
    finished_at = now(),
    error_message = sqlc.arg('error_message')::text
where image_request.id = sqlc.arg('image_request_id')
    and finished_at is null;

-- name: RecordImageRequestSuccess :execresult
update showtime.image_request set
    finished_at = now()
where image_request.id = sqlc.arg('image_request_id')
    and finished_at is null;

-- name: RecordImage :exec
insert into showtime.image (
    image_request_id,
    index,
    url
) values (
    sqlc.arg('image_request_id'),
    sqlc.arg('index'),
    sqlc.arg('url')
);

-- name: GetImagesForRequest :many
select
    image.url
from showtime.image
where image.image_request_id = sqlc.arg('image_request_id')
order by image.index;
