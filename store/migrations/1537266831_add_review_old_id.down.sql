BEGIN;

ALTER TABLE review_event DROP COLUMN old_internal_id;

COMMIT;
