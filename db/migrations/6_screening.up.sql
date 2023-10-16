begin;

drop table showtime.tape_change;

create table showtime.screening (
    broadcast_id integer not null,
    tape_id      integer not null,
    started_at   timestamptz not null default now(),
    ended_at     timestamptz
);

alter table showtime.screening
    add constraint screening_broadcast_id_fk
    foreign key (broadcast_id) references showtime.broadcast (id);

comment on table showtime.screening is
    'Records the fact that a particular tape was played during a broadcast.';
comment on column showtime.screening.broadcast_id is
    'ID of the broadcast that was live at the time the screening started.';
comment on column showtime.screening.tape_id is
    'ID of the tape that was screened.';
comment on column showtime.screening.started_at is
    'Time at which the screening started.';
comment on column showtime.screening.ended_at is
    'Time at which the screening ended, if it''s not stil ongoing.';

create index screening_tape_id_index on showtime.screening (tape_id);

commit;
