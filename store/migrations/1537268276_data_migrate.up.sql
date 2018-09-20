BEGIN;

/* ReviewEvent: fill OldInternalID */
UPDATE review_event SET old_internal_id = internal_id WHERE old_internal_id = '';

/* Load uuid-ossp extension to be able to generate uuid */
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

/* ReviewTarget: create for old event */
INSERT INTO review_target (id, "provider", internal_id, repository_id, "number", created_at, updated_at)
SELECT uuid_generate_v4(), "provider", internal_id, repository_id, "number", created_at, updated_at FROM review_event
WHERE review_target_id IS null;

/* ReviewEvent: set review_target_id */
UPDATE review_event
SET review_target_id = review_target.id
FROM review_target
WHERE
    review_event.provider = review_target.provider AND
    review_event.internal_id = review_target.internal_id AND
    review_event.repository_id = review_target.repository_id AND
    review_event.number = review_target.number AND
    review_event.review_target_id IS null;

/* add default values to old columns, because we don't write them anymore and they can't be null */
ALTER TABLE ONLY review_event ALTER COLUMN "provider" SET DEFAULT '';
ALTER TABLE ONLY review_event ALTER COLUMN "internal_id" SET DEFAULT '';
ALTER TABLE ONLY review_event ALTER COLUMN "repository_id" SET DEFAULT 0;
ALTER TABLE ONLY review_event ALTER COLUMN "number" SET DEFAULT 0;

COMMIT;
