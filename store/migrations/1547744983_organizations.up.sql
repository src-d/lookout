BEGIN;

CREATE TABLE organization (
	id uuid NOT NULL PRIMARY KEY,
	provider text NOT NULL,
	internal_id text NOT NULL,
	config text NOT NULL
);


COMMIT;
