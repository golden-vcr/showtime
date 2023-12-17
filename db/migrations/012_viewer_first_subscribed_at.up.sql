begin;

alter table showtime.viewer
    add column first_subscribed_at timestamptz;

comment on column showtime.viewer.first_subscribed_at is
    'Timestamp when the user first became a subscriber of GoldenVCR on Twitch, if '
    'ever. Does not guarantee that the user still has an ongoing subscription; and '
    'does not distinguish gift subs from subs purchased by the viewer directly.';

commit;
