BEGIN;

ALTER TABLE review_event ADD COLUMN provider text NOT NULL;

ALTER TABLE review_event ADD COLUMN internal_id text NOT NULL;

ALTER TABLE review_event ADD COLUMN repository_id bigint NOT NULL;

ALTER TABLE review_event ADD COLUMN number bigint NOT NULL;

COMMIT;
