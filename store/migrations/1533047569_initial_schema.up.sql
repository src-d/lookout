BEGIN;

CREATE TABLE comment (
	id uuid NOT NULL PRIMARY KEY,
	file text NOT NULL,
	line integer NOT NULL,
	text text NOT NULL,
	confidence bigint NOT NULL
);


CREATE TABLE review_event (
	id uuid NOT NULL PRIMARY KEY,
	status text NOT NULL,
	provider text NOT NULL,
	internal_id text NOT NULL,
	created_at timestamptz NOT NULL,
	updated_at timestamptz NOT NULL,
	is_mergeable boolean NOT NULL,
	source jsonb NOT NULL,
	merge jsonb NOT NULL,
	configuration jsonb NOT NULL,
	base jsonb NOT NULL,
	head jsonb NOT NULL
);


COMMIT;
