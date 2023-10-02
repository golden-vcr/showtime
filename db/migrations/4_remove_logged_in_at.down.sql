begin;

alter table showtime.viewer
    add column first_logged_in_at timestamptz,
    add column last_logged_in_at timestamptz;

comment on column showtime.viewer.first_logged_in_at is
    'Timestamp when the user first logged in at goldenvcr.com, if if ever.';
comment on column showtime.viewer.last_logged_in_at is
    'Timestamp when the user most recently logged in at goldenvcr.com, if ever.';

commit;
