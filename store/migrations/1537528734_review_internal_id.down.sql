BEGIN;

ALTER TABLE review_event DROP COLUMN internal_id;

COMMIT;
