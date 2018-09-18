BEGIN;

/* use default as start of the time, it will affect only old rows */

ALTER TABLE comment ADD COLUMN created_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00';

ALTER TABLE comment ADD COLUMN updated_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00';

ALTER TABLE review_target ADD COLUMN created_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00';

ALTER TABLE review_target ADD COLUMN updated_at timestamptz NOT NULL DEFAULT '1970-01-01 00:00:00+00';

COMMIT;
