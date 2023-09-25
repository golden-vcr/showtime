begin;

create table showtime.tape_change (
    created_at timestamptz not null default now(),
    tape_id    text not null
);

comment on table showtime.tape_change is
    'Records the fact that a new tape was set as current (or the current tape was '
    'cleared).';
comment on column showtime.tape_change.created_at is
    'Timestamp at which the tape change occurred.';
comment on column showtime.tape_change.tape_id is
    'ID of the tape that was set as current (if non-empty), or empty string to '
    'indicate that the current tape was cleared.';

create index tape_change_created_at_idx on showtime.tape_change (created_at);
create index tape_change_tape_id_idex on showtime.tape_change (tape_id);

commit;
