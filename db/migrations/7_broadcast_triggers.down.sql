begin;

drop trigger notify_on_broadcast_change on showtime.broadcast;
drop trigger notify_on_screening_change on showtime.screening;

drop function emit_broadcast_change_notification;
drop function emit_screening_change_notification;

commit;
