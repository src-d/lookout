BEGIN;

ALTER TABLE review_event ADD COLUMN repository_id bigint NOT NULL;

COMMIT;
