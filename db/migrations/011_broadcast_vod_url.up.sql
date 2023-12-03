begin;

alter table showtime.broadcast
    add column vod_url text;

comment on column showtime.broadcast.vod_url is
    'Absolute URL to a page where the recording of this broadcast can be viewed, if '
    'available.';

commit;
