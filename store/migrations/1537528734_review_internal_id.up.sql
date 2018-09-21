BEGIN;

ALTER TABLE review_event ADD COLUMN internal_id text NOT NULL default '';

COMMIT;
