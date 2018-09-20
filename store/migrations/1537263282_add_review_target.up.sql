BEGIN;

CREATE TABLE review_target (
	id uuid NOT NULL PRIMARY KEY,
	provider text NOT NULL,
	internal_id text NOT NULL,
	repository_id bigint NOT NULL,
	number bigint NOT NULL
);


ALTER TABLE review_event ADD COLUMN review_target_id uuid REFERENCES review_target(id);

COMMIT;
