BEGIN;

ALTER TABLE review_event ADD COLUMN merge jsonb NOT NULL;

COMMIT;
