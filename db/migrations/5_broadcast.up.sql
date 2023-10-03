begin;

create table showtime.broadcast (
    id         serial primary key,
    started_at timestamptz not null default now(),
    ended_at   timestamptz
);

comment on table showtime.broadcast is
    'Record of a broadcast that occurred (or is occurring) on the GoldenVCR Twitch '
    'channel.';
comment on column showtime.broadcast.id is
    'Serial ID used to correlate other records with this broadcast.';
comment on column showtime.broadcast.started_at is
    'Time at which this broadcast first started.';
comment on column showtime.broadcast.ended_at is
    'Time at which the broadcast ended, if it''s not still live. To account for the '
    'possibility of brief disruptions in internet service (or Twitch availability), '
    'it''s possible to resume a broadcast once it''s ended: a non-NULL ended_at '
    'timestamp does not definitively indicate that broadcast is done for good.';

commit;
