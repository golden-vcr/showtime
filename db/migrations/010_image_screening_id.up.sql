begin;

alter table showtime.image_request
    add column screening_id uuid;

comment on column showtime.image_request.screening_id is
    'ID of the screening that was active when the image request was submitted; may be '
    'null if the request was submitted while no tape was active.';

update showtime.image_request
    set screening_id = (
        select id from showtime.screening
            where image_request.created_at between screening.started_at and screening.ended_at
            limit 1
    );

commit;
