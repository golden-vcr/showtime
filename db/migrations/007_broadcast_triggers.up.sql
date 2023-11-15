begin;

create function emit_broadcast_change_notification() returns trigger as $trigger$
begin
    perform pg_notify('showtime', json_build_object(
        'type', 'broadcast',
        'data', json_build_object(
            'id', NEW.id,
            'started_at', NEW.started_at,
            'ended_at', NEW.ended_at
        )
    )::text);
    return NEW;
end;
$trigger$ language plpgsql;

create function emit_screening_change_notification() returns trigger as $trigger$
begin
    perform pg_notify('showtime', json_build_object(
        'type', 'screening',
        'data', json_build_object(
            'broadcast_id', NEW.broadcast_id,
            'tape_id', NEW.tape_id,
            'started_at', NEW.started_at,
            'ended_at', NEW.ended_at
        )
    )::text);
    return NEW;
end;
$trigger$ language plpgsql;

create trigger notify_on_broadcast_change
    after insert or update on showtime.broadcast
    for each row execute procedure emit_broadcast_change_notification();

create trigger notify_on_screening_change
    after insert or update on showtime.screening
    for each row execute procedure emit_screening_change_notification();

commit;
