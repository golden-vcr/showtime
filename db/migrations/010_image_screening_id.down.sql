begin;

alter table showtime.image_request
    drop column screening_id;

commit;
