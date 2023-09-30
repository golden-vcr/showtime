begin;

create table showtime.viewer (
    twitch_user_id      text primary key,
    twitch_display_name text not null,
    first_logged_in_at  timestamptz,
    last_logged_in_at   timestamptz,
    first_followed_at   timestamptz
);

comment on table showtime.viewer is
    'Details about a user who has interacted with Golden VCR at some point, either '
    'directly via goldenvcr.com or via Twitch.';
comment on column showtime.viewer.twitch_user_id is
    'Text-formatted integer identifying this user in the Twitch API.';
comment on column showtime.viewer.twitch_display_name is
    'Last known username by which this user was known, formatted for display.';
comment on column showtime.viewer.first_logged_in_at is
    'Timestamp when the user first logged in at goldenvcr.com, if if ever.';
comment on column showtime.viewer.last_logged_in_at is
    'Timestamp when the user most recently logged in at goldenvcr.com, if ever.';
comment on column showtime.viewer.first_followed_at is
    'Timestamp when the user first followed GoldenVCR on twitch, if ever.';

commit;
