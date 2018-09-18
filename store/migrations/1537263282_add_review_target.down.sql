BEGIN;

DROP TABLE review_target;

ALTER TABLE review_event DROP COLUMN review_target_id;

COMMIT;
