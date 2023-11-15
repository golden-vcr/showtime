begin;

-- Add an ID column to the screening table, with no constraints initially
alter table showtime.screening
    add column id uuid;

comment on column showtime.screening.id is
    'Unique ID for this screening; used chiefly to associate other data with this '
    'screening.';

-- Add a random UUID to all existing screenings
update showtime.screening set id = gen_random_uuid();

-- Now that all of our existing screenings have an ID, make that column the primary key
-- and establish a default value
alter table showtime.screening
    add primary key (id);

alter table showtime.screening
    alter column id set default gen_random_uuid();

commit;
