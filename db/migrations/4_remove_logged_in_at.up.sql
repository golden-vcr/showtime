begin;

alter table showtime.viewer
    drop column first_logged_in_at,
    drop column last_logged_in_at;

commit;
