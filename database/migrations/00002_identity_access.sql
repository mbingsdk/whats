-- +goose Up
-- Global identity is an explicit auth-service exception; tenant data still FORCE RLS.
GRANT USAGE ON SCHEMA app TO waba_identity;
ALTER TABLE app.users ADD COLUMN verified_at timestamptz;
CREATE TABLE app.identity_bootstrap (singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), organization_id uuid NOT NULL REFERENCES app.organizations(id), user_id uuid NOT NULL REFERENCES app.users(id), created_at timestamptz NOT NULL DEFAULT now());
GRANT SELECT,INSERT ON app.identity_bootstrap TO waba_identity;
CREATE TABLE app.password_identities (
 user_id uuid PRIMARY KEY REFERENCES app.users(id), password_hash text NOT NULL, changed_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.sessions (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES app.users(id),
 token_digest text NOT NULL UNIQUE, csrf_digest text NOT NULL, auth_revision bigint NOT NULL,
 organization_id uuid REFERENCES app.organizations(id), access_revision bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL, idle_expires_at timestamptz NOT NULL,
 reauth_at timestamptz NOT NULL, mfa_at timestamptz, revoked_at timestamptz,
 CHECK(idle_expires_at <= expires_at)
);
CREATE INDEX sessions_user ON app.sessions(user_id,revoked_at);
CREATE INDEX sessions_org ON app.sessions(organization_id,user_id);
CREATE TABLE app.auth_challenges (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES app.users(id),
 kind text NOT NULL CHECK(kind IN ('EMAIL_VERIFY','PASSWORD_RESET','LOGIN_MFA')),
 token_digest text NOT NULL UNIQUE, expires_at timestamptz NOT NULL,
 consumed_at timestamptz, revoked_at timestamptz, attempts integer NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX challenges_user ON app.auth_challenges(user_id,kind,expires_at);
CREATE TABLE app.auth_factors (
 user_id uuid PRIMARY KEY REFERENCES app.users(id), secret_ciphertext bytea NOT NULL,
 confirmed_at timestamptz, last_step bigint NOT NULL DEFAULT -1, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE app.auth_recovery_codes (
 id uuid PRIMARY KEY, user_id uuid NOT NULL REFERENCES app.users(id), digest text NOT NULL UNIQUE,
 consumed_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX recovery_user ON app.auth_recovery_codes(user_id);
CREATE TABLE app.login_limits (
 key text PRIMARY KEY, window_at timestamptz NOT NULL, attempts integer NOT NULL, expires_at timestamptz NOT NULL
);
CREATE INDEX limits_expiry ON app.login_limits(expires_at);
CREATE TABLE app.identity_audit (
 id uuid PRIMARY KEY, user_id uuid REFERENCES app.users(id), action text NOT NULL,
 resource_id uuid, request_id text NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX identity_audit_user ON app.identity_audit(user_id,created_at);
CREATE TABLE app.permissions (
 key text PRIMARY KEY, sensitive boolean NOT NULL DEFAULT false
);
INSERT INTO app.permissions(key,sensitive) VALUES
 ('organization.view',false),('members.view',false),('members.invite',true),('members.manage',true),
 ('teams.view',false),('teams.manage',true),('roles.view',false),('roles.manage',true),
 ('roles.assign',true),('owners.manage',true),('audit.view',false);
CREATE TABLE app.roles (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 name text NOT NULL, revision bigint NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(organization_id,id), UNIQUE(organization_id,name)
);
CREATE TABLE app.role_permissions (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 role_id uuid NOT NULL, permission_key text NOT NULL REFERENCES app.permissions(key),
 FOREIGN KEY(organization_id,role_id) REFERENCES app.roles(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,role_id,permission_key)
);
CREATE TABLE app.teams (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 name text NOT NULL, archived_at timestamptz, revision bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(organization_id,id)
);
CREATE UNIQUE INDEX teams_name ON app.teams(organization_id,lower(name)) WHERE archived_at IS NULL;
CREATE TABLE app.team_members (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 team_id uuid NOT NULL, member_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,team_id) REFERENCES app.teams(organization_id,id),
 FOREIGN KEY(organization_id,member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,team_id,member_id)
);
CREATE INDEX team_members_member ON app.team_members(organization_id,member_id);
CREATE TABLE app.member_roles (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 member_id uuid NOT NULL, role_id uuid NOT NULL, scope_kind text NOT NULL CHECK(scope_kind IN ('ORG','TEAM','SELF')),
 team_id uuid, expires_at timestamptz, created_at timestamptz NOT NULL DEFAULT now(),
 CHECK((scope_kind='TEAM')=(team_id IS NOT NULL)),
 FOREIGN KEY(organization_id,member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,role_id) REFERENCES app.roles(organization_id,id),
 FOREIGN KEY(organization_id,team_id) REFERENCES app.teams(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE NULLS NOT DISTINCT(organization_id,member_id,role_id,scope_kind,team_id)
);
CREATE INDEX member_roles_team ON app.member_roles(organization_id,team_id);
CREATE INDEX member_roles_role ON app.member_roles(organization_id,role_id);
CREATE TABLE app.team_roles (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 team_id uuid NOT NULL, role_id uuid NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,team_id) REFERENCES app.teams(organization_id,id),
 FOREIGN KEY(organization_id,role_id) REFERENCES app.roles(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,team_id,role_id)
);
CREATE INDEX team_roles_role ON app.team_roles(organization_id,role_id);
CREATE TABLE app.invitations (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 canonical_email text NOT NULL CHECK(canonical_email=lower(canonical_email)),
 token_digest text NOT NULL UNIQUE, inviter_member_id uuid NOT NULL,
 state text NOT NULL DEFAULT 'PENDING' CHECK(state IN ('PENDING','ACCEPTED','REVOKED','EXPIRED')),
 expires_at timestamptz NOT NULL, revision bigint NOT NULL DEFAULT 1,
 accepted_member_id uuid, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,inviter_member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,accepted_member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id)
);
CREATE UNIQUE INDEX pending_invitation_email ON app.invitations(organization_id,canonical_email) WHERE state='PENDING';
CREATE INDEX invitation_accepted ON app.invitations(organization_id,accepted_member_id);
CREATE INDEX invitation_inviter ON app.invitations(organization_id,inviter_member_id);
CREATE TABLE app.invitation_roles (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 invitation_id uuid NOT NULL, role_id uuid NOT NULL,
 FOREIGN KEY(organization_id,invitation_id) REFERENCES app.invitations(organization_id,id),
 FOREIGN KEY(organization_id,role_id) REFERENCES app.roles(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,invitation_id,role_id)
);
CREATE INDEX invitation_roles_role ON app.invitation_roles(organization_id,role_id);
CREATE TABLE app.audit_log (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 actor_user_id uuid REFERENCES app.users(id), actor_member_id uuid, action text NOT NULL,
 resource_id uuid, request_id text NOT NULL, actor_kind text NOT NULL DEFAULT 'USER', actor_label_snapshot text, change_digest text NOT NULL DEFAULT '', resource_revision bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,actor_member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id)
);
CREATE INDEX audit_member ON app.audit_log(organization_id,actor_member_id);
CREATE INDEX audit_user ON app.audit_log(actor_user_id);
CREATE INDEX audit_org ON app.audit_log(organization_id,created_at,id);
CREATE TABLE app.mail_deliveries (
 id uuid PRIMARY KEY, user_id uuid REFERENCES app.users(id), organization_id uuid REFERENCES app.organizations(id),
 invitation_id uuid, challenge_id uuid REFERENCES app.auth_challenges(id),
 purpose text NOT NULL CHECK(purpose IN ('INVITATION','EMAIL_VERIFY','PASSWORD_RESET','SECURITY_NOTICE')),
 source_id uuid NOT NULL, template_version smallint NOT NULL DEFAULT 1 CHECK(template_version=1), payload_ciphertext bytea,
 state text NOT NULL DEFAULT 'QUEUED' CHECK(state IN ('QUEUED','SENDING','SMTP_ACCEPTED','RETRY_WAIT','FAILED','EXPIRED','CANCELLED')),
 deadline_at timestamptz NOT NULL, next_attempt_at timestamptz NOT NULL DEFAULT now(),
 attempt_count integer NOT NULL DEFAULT 0, lease_token uuid, lease_until timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), terminal_at timestamptz,
 FOREIGN KEY(organization_id,invitation_id) REFERENCES app.invitations(organization_id,id),
 CHECK((purpose='INVITATION')=(invitation_id IS NOT NULL)),
 CHECK(invitation_id IS NULL OR organization_id IS NOT NULL), UNIQUE(purpose,source_id)
);
CREATE INDEX mail_due ON app.mail_deliveries(state,next_attempt_at);
CREATE INDEX mail_challenge ON app.mail_deliveries(challenge_id);
CREATE INDEX mail_user ON app.mail_deliveries(user_id);
CREATE INDEX mail_invitation ON app.mail_deliveries(organization_id,invitation_id);
CREATE TABLE app.mail_attempts (
 id uuid PRIMARY KEY, delivery_id uuid NOT NULL REFERENCES app.mail_deliveries(id),
 attempt_no integer NOT NULL, result text CHECK(result IN ('SMTP_ACCEPTED','REJECTED','UNKNOWN')),
 error_code text, started_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz,
 UNIQUE(delivery_id,attempt_no)
);
-- +goose StatementBegin
DO $$
DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['roles','role_permissions','teams','team_members','member_roles','team_roles','invitations','invitation_roles','audit_log']
 LOOP
  EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE app.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY tenant_scope ON app.%I TO waba_runtime,waba_identity USING (organization_id=nullif(current_setting(''app.organization_id'',true),'''')::uuid) WITH CHECK (organization_id=nullif(current_setting(''app.organization_id'',true),'''')::uuid)',t);
  IF t <> 'audit_log' THEN
   EXECUTE format('GRANT SELECT,INSERT,UPDATE,DELETE ON app.%I TO waba_runtime,waba_identity',t);
  ELSE
   EXECUTE format('GRANT SELECT,INSERT ON app.%I TO waba_runtime,waba_identity',t);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
GRANT SELECT ON app.permissions TO waba_runtime,waba_identity;
GRANT SELECT,INSERT,UPDATE ON app.users TO waba_identity;
GRANT SELECT,INSERT,UPDATE,DELETE ON app.password_identities,app.sessions,app.auth_challenges,app.auth_factors,app.auth_recovery_codes,app.login_limits,app.mail_deliveries,app.mail_attempts TO waba_identity;
GRANT SELECT,INSERT ON app.identity_audit TO waba_identity;
GRANT SELECT,INSERT,UPDATE ON app.organizations TO waba_identity;
GRANT SELECT,INSERT,UPDATE,DELETE ON app.organization_members TO waba_identity;
CREATE POLICY identity_org_scope ON app.organizations TO waba_identity USING(id=nullif(current_setting('app.organization_id',true),'')::uuid) WITH CHECK(id=nullif(current_setting('app.organization_id',true),'')::uuid);
CREATE POLICY identity_member_scope ON app.organization_members TO waba_identity USING(organization_id=nullif(current_setting('app.organization_id',true),'')::uuid) WITH CHECK(organization_id=nullif(current_setting('app.organization_id',true),'')::uuid);
SET LOCAL ROLE waba_authorizer;
GRANT EXECUTE ON FUNCTION app.authorize_membership(uuid,uuid) TO waba_identity;
RESET ROLE;
GRANT SELECT ON public.goose_db_version TO waba_identity;
-- Narrow pre-context directory routing. No runtime table grants or role membership.
GRANT SELECT ON app.invitations TO waba_authorizer;
CREATE POLICY invitation_router ON app.invitations TO waba_authorizer USING(true) WITH CHECK(false);
-- +goose StatementBegin
CREATE FUNCTION app.invitation_organization(p_digest text) RETURNS uuid LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT organization_id FROM app.invitations WHERE token_digest=p_digest AND state='PENDING' AND expires_at>now()
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.user_organizations(p_user uuid) RETURNS TABLE(id uuid,name text,member_id uuid,access_revision bigint) LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT o.id,o.name,m.id,m.access_revision FROM app.organizations o JOIN app.organization_members m ON m.organization_id=o.id
 JOIN app.users u ON u.id=m.user_id WHERE u.id=p_user AND u.global_status='ACTIVE' AND m.status='ACTIVE' AND o.status='ACTIVE'
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.authorize_membership_write(p_user uuid,p_org uuid) RETURNS uuid LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
DECLARE result uuid;
BEGIN
 PERFORM 1 FROM app.users WHERE id=p_user AND global_status='ACTIVE' FOR SHARE;
 IF NOT FOUND THEN RETURN NULL; END IF;
 PERFORM 1 FROM app.organizations WHERE id=p_org AND status='ACTIVE' FOR UPDATE;
 IF NOT FOUND THEN RETURN NULL; END IF;
 SELECT id INTO result FROM app.organization_members WHERE user_id=p_user AND organization_id=p_org AND status='ACTIVE' FOR SHARE;
 RETURN result;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.invitation_organization(text),app.user_organizations(uuid),app.authorize_membership_write(uuid,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.invitation_organization(text),app.user_organizations(uuid),app.authorize_membership_write(uuid,uuid) TO waba_identity;
GRANT CREATE ON SCHEMA app TO waba_authorizer;
ALTER FUNCTION app.invitation_organization(text) OWNER TO waba_authorizer;
ALTER FUNCTION app.user_organizations(uuid) OWNER TO waba_authorizer;
ALTER FUNCTION app.authorize_membership_write(uuid,uuid) OWNER TO waba_authorizer;
REVOKE CREATE ON SCHEMA app FROM waba_authorizer;
-- Completed mail attempts cannot be rewritten by application credentials.
-- +goose StatementBegin
CREATE FUNCTION app.protect_mail_attempt() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF OLD.finished_at IS NOT NULL THEN RAISE EXCEPTION 'Terminal mail attempt is immutable'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_mail_attempt BEFORE UPDATE OR DELETE ON app.mail_attempts FOR EACH ROW EXECUTE FUNCTION app.protect_mail_attempt();
REVOKE ALL ON FUNCTION app.protect_mail_attempt() FROM PUBLIC;
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Forward repair required'; END $$;
-- +goose StatementEnd
