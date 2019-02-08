BEGIN;

ALTER TABLE push_event ADD COLUMN organization_id text NOT NULL;

COMMIT;
