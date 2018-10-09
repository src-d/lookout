BEGIN;

ALTER TABLE review_event ADD COLUMN old_internal_id text NOT NULL;

COMMIT;
