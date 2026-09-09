-- +goose Up
CREATE SCHEMA app AUTHORIZATION waba_migrator;
REVOKE ALL ON SCHEMA app FROM PUBLIC;
GRANT USAGE ON SCHEMA app TO waba_runtime, waba_authorizer;
ALTER DEFAULT PRIVILEGES IN SCHEMA app REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC;

CREATE TABLE app.users (
 id uuid PRIMARY KEY,
 canonical_email text NOT NULL UNIQUE CHECK (canonical_email = lower(canonical_email)),
 display_name text NOT NULL,
 global_status text NOT NULL DEFAULT 'ACTIVE' CHECK (global_status IN ('ACTIVE','SUSPENDED')),
 auth_revision bigint NOT NULL DEFAULT 1 CHECK (auth_revision > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.organizations (
 id uuid PRIMARY KEY,
 name text NOT NULL,
 slug text NOT NULL UNIQUE,
 timezone text NOT NULL,
 status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED')),
 revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.organization_members (
 id uuid PRIMARY KEY,
 organization_id uuid NOT NULL REFERENCES app.organizations(id),
 user_id uuid NOT NULL REFERENCES app.users(id),
 status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','DEACTIVATED')),
 access_revision bigint NOT NULL DEFAULT 1 CHECK (access_revision > 0),
 activated_at timestamptz,
 deactivated_at timestamptz,
 revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(organization_id,id),
 UNIQUE(organization_id,user_id)
);
CREATE INDEX organization_members_user_idx ON app.organization_members(user_id);
CREATE INDEX organization_members_status_idx ON app.organization_members(organization_id,status);
ALTER TABLE app.organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.organizations FORCE ROW LEVEL SECURITY;
ALTER TABLE app.organization_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.organization_members FORCE ROW LEVEL SECURITY;
CREATE POLICY organization_scope ON app.organizations TO waba_runtime
 USING (id = nullif(current_setting('app.organization_id',true),'')::uuid);
CREATE POLICY member_scope ON app.organization_members TO waba_runtime
 USING (organization_id = nullif(current_setting('app.organization_id',true),'')::uuid)
 WITH CHECK (organization_id = nullif(current_setting('app.organization_id',true),'')::uuid);
CREATE POLICY authorization_org_read ON app.organizations TO waba_authorizer USING (true) WITH CHECK (false);
CREATE POLICY authorization_member_read ON app.organization_members TO waba_authorizer USING (true) WITH CHECK (false);
GRANT SELECT ON app.users, app.organizations, app.organization_members TO waba_authorizer;
GRANT UPDATE(id) ON app.users, app.organizations, app.organization_members TO waba_authorizer;
GRANT SELECT ON app.organizations TO waba_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON app.organization_members TO waba_runtime;
-- Only this narrowly defined function can inspect memberships before tenant context.
-- The caller must supply a server-authenticated user, never an untrusted user header.
-- +goose StatementBegin
CREATE FUNCTION app.authorize_membership(p_user uuid,p_org uuid) RETURNS uuid
 LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog, app
 AS $fn$
 SELECT m.id FROM app.organization_members m
 JOIN app.users u ON u.id=m.user_id
 JOIN app.organizations o ON o.id=m.organization_id
 WHERE m.user_id=p_user AND m.organization_id=p_org
 AND m.status='ACTIVE' AND u.global_status='ACTIVE' AND o.status='ACTIVE'
 FOR SHARE OF m,u,o
 $fn$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.authorize_membership(uuid,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.authorize_membership(uuid,uuid) TO waba_runtime;
GRANT CREATE ON SCHEMA app TO waba_authorizer;
ALTER FUNCTION app.authorize_membership(uuid,uuid) OWNER TO waba_authorizer;
REVOKE CREATE ON SCHEMA app FROM waba_authorizer;
GRANT SELECT ON public.goose_db_version TO waba_runtime;

-- +goose Down
-- Forward-only production convention; restore/forward repair needs a reviewed plan.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Destructive down migration disabled; use reviewed forward repair'; END $$;
-- +goose StatementEnd
