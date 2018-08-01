BEGIN;

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


CREATE TABLE comment (
	id uuid NOT NULL PRIMARY KEY,
	review_event_id uuid REFERENCES review_event(id),
	file text NOT NULL,
	line integer NOT NULL,
	text text NOT NULL,
	confidence bigint NOT NULL
);


CREATE TABLE push_event (
	id uuid NOT NULL PRIMARY KEY,
	status text NOT NULL,
	provider text NOT NULL,
	internal_id text NOT NULL,
	created_at timestamptz NOT NULL,
	commits bigint NOT NULL,
	distinct_commits bigint NOT NULL,
	configuration jsonb NOT NULL,
	base jsonb NOT NULL,
	head jsonb NOT NULL
);


COMMIT;
