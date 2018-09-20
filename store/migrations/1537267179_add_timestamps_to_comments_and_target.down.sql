BEGIN;

ALTER TABLE comment DROP COLUMN created_at;

ALTER TABLE comment DROP COLUMN updated_at;

ALTER TABLE review_target DROP COLUMN created_at;

ALTER TABLE review_target DROP COLUMN updated_at;

COMMIT;
