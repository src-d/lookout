BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

UPDATE review_event AS e
SET internal_id=encode(digest(concat(t.provider, '|', t.internal_id, '|', e.head::json->>'hash'), 'sha1'), 'hex')
FROM review_target AS t
WHERE e.internal_id='' AND t.id = e.review_target_id;

COMMIT;
