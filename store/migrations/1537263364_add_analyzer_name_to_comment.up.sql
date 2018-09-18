BEGIN;

ALTER TABLE comment ADD COLUMN analyzer text NOT NULL default '';

COMMIT;
