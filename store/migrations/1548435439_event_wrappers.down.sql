BEGIN;

ALTER TABLE push_event DROP COLUMN organization_id;

COMMIT;
