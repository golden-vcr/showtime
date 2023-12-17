begin;

alter table showtime.viewer
    drop column first_subscribed_at;

commit;
