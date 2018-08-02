BEGIN;

ALTER TABLE review_event ADD COLUMN number bigint NOT NULL;

COMMIT;
