begin;

create table showtime.image_request (
    id                  uuid primary key,
    twitch_user_id      text not null,

    subject_noun_clause text not null,
    prompt              text not null,

    created_at          timestamptz not null default now(),
    finished_at         timestamptz,
    error_message       text
);

comment on table showtime.image_request is
    'Records the fact that a user requested that images be generated, with their '
    'chosen prompt, to be overlaid on the video during the stream.';
comment on column showtime.image_request.id is
    'Globally unique identifier for this request.';
comment on column showtime.image_request.twitch_user_id is
    'ID of the Twitch user that initiated this request.';
comment on column showtime.image_request.subject_noun_clause is
    'Noun clause describing the desired subject of the image, e.g. "a cardboard box", '
    '"several large turkeys", "the concept of love".';
comment on column showtime.image_request.prompt is
    'The complete prompt that was submitted in order to initiate image generation, '
    'e.g. "a ghostly image of several large turkeys, with glitchy VHS artifacts, dark '
    'background".';
comment on column showtime.image_request.created_at is
    'Timestamp indicating when the request was submitted.';
comment on column showtime.image_request.finished_at is
    'Timestamp indicating when we received a response for the image generation '
    'request, whether successful or not. If NULL, the request is still being processed '
    'and images are not ready yet.';
comment on column showtime.image_request.error_message is
    'Error message describing why the request completed unsuccessfully. If NULL and '
    'finished_at is not NULL, the request completed successfully.';

alter table showtime.image_request
    add constraint image_request_viewer_id_fk
    foreign key (twitch_user_id) references showtime.viewer (twitch_user_id);

create table showtime.image (
    image_request_id uuid not null,
    index            integer not null,
    url              text not null
);

comment on table showtime.image is
    'Record of an image that was successfully generated from a user-submitted image '
    'request. An image request may result in multiple images. Images are ordered by '
    'index, matching the array in which they were returned by the image generation '
    'API.';
comment on column showtime.image.image_request_id is
    'ID of the image_request record associated with this image.';
comment on column showtime.image.index is
    'Sequential, zero-indexed position of this image in the original results array.';
comment on column showtime.image.url is
    'URL indicating where the image has been uploaded for long-term storage, so that '
    'it can be displayed in client applications.';

alter table showtime.image
    add constraint image_request_id_fk
    foreign key (image_request_id) references showtime.image_request (id);

alter table showtime.image
    add constraint image_request_id_index_unique
    unique (image_request_id, index);

commit;
