BEGIN;

ALTER TABLE review_event DROP COLUMN provider;

ALTER TABLE review_event DROP COLUMN internal_id;

ALTER TABLE review_event DROP COLUMN repository_id;

ALTER TABLE review_event DROP COLUMN number;

COMMIT;
