/* go-kallax created a id uuid primary key, but the application logic acts
as if the primary key is (provider, internal_id) */

CREATE UNIQUE INDEX organization_composite_pkey
	ON organization (provider, internal_id);
