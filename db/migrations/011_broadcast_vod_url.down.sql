begin;

alter table showtime.broadcast
    drop column vod_url;

commit;
