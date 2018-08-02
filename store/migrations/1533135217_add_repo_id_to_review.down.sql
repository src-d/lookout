BEGIN;

ALTER TABLE review_event DROP COLUMN repository_id;

COMMIT;
