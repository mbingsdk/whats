-- +goose Up
CREATE TABLE app.meta_apps (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 external_id text NOT NULL UNIQUE, callback_key uuid NOT NULL UNIQUE,
 graph_version text NOT NULL CHECK(graph_version='v26.0'), configured_waba_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 last_sync_at timestamptz, last_error text, credential_state text NOT NULL DEFAULT 'UNTESTED',
 challenge_verified_at timestamptz, last_webhook_at timestamptz,
 UNIQUE(organization_id,id)
);
CREATE TABLE app.wabas (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), app_id uuid NOT NULL,
 external_id text NOT NULL UNIQUE, name text NOT NULL, timezone_id text NOT NULL,
 subscribed boolean NOT NULL, lifecycle text NOT NULL CHECK(lifecycle IN ('PRESENT','NOT_OBSERVED')),
 graph_version text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), synced_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,app_id) REFERENCES app.meta_apps(organization_id,id), UNIQUE(organization_id,id)
);
CREATE TABLE app.phone_numbers (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), waba_id uuid NOT NULL,
 external_id text NOT NULL UNIQUE, name text NOT NULL, display_number text NOT NULL,
 quality text NOT NULL, platform text NOT NULL, code_verification_status text NOT NULL,
 lifecycle text NOT NULL CHECK(lifecycle IN ('PRESENT','NOT_OBSERVED')), graph_version text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), synced_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,waba_id) REFERENCES app.wabas(organization_id,id), UNIQUE(organization_id,id)
);
CREATE TABLE app.business_profiles (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), phone_id uuid NOT NULL,
 fields jsonb NOT NULL, graph_version text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), synced_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,phone_id) REFERENCES app.phone_numbers(organization_id,id), UNIQUE(organization_id,id), UNIQUE(organization_id,phone_id)
);
CREATE TABLE app.asset_sync_runs (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), app_id uuid NOT NULL,
 state text NOT NULL DEFAULT 'QUEUED' CHECK(state IN ('QUEUED','RUNNING','SUCCEEDED','FAILED')),
 request_id text NOT NULL, requested_by uuid NOT NULL REFERENCES app.users(id), attempt_count integer NOT NULL DEFAULT 0,
 lease_token uuid, lease_until timestamptz, next_attempt_at timestamptz NOT NULL DEFAULT now(),
 error_code text, graph_version text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz,
 FOREIGN KEY(organization_id,app_id) REFERENCES app.meta_apps(organization_id,id), UNIQUE(organization_id,id)
);
CREATE UNIQUE INDEX one_active_meta_sync ON app.asset_sync_runs(organization_id,app_id) WHERE state IN ('QUEUED','RUNNING');
CREATE TABLE app.webhook_events (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), app_id uuid NOT NULL,
 raw_digest text NOT NULL, raw_ciphertext bytea, received_at timestamptz NOT NULL DEFAULT now(),
 retain_until timestamptz NOT NULL DEFAULT now()+interval '7 days', request_id text NOT NULL,
 state text NOT NULL DEFAULT 'QUEUED' CHECK(state IN ('QUEUED','PROCESSING','RETRY_WAIT','PROCESSED','UNKNOWN','INVALID','QUARANTINED','DEAD_LETTER')),
 event_class text NOT NULL DEFAULT 'UNCLASSIFIED', parser_version integer NOT NULL DEFAULT 1,
 attempt_count integer NOT NULL DEFAULT 0, generation integer NOT NULL DEFAULT 0,
 next_attempt_at timestamptz NOT NULL DEFAULT now(), lease_token uuid, lease_until timestamptz,
 last_error text, processed_at timestamptz,
 FOREIGN KEY(organization_id,app_id) REFERENCES app.meta_apps(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,app_id,raw_digest)
);
CREATE INDEX webhook_due ON app.webhook_events(organization_id,state,next_attempt_at);
CREATE TABLE app.webhook_facts (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), event_id uuid NOT NULL,
 waba_id uuid NOT NULL, phone_id uuid, event_class text NOT NULL,
 effect_key text NOT NULL, uncertain_identity boolean NOT NULL, observed_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,event_id) REFERENCES app.webhook_events(organization_id,id),
 FOREIGN KEY(organization_id,waba_id) REFERENCES app.wabas(organization_id,id),
 FOREIGN KEY(organization_id,phone_id) REFERENCES app.phone_numbers(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,effect_key)
);
CREATE TABLE app.webhook_processing_attempts (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), event_id uuid NOT NULL,
 generation integer NOT NULL, attempt_no integer NOT NULL, result text NOT NULL, error_code text,
 parser_version integer NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,event_id) REFERENCES app.webhook_events(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,event_id,generation,attempt_no)
);
CREATE TABLE app.webhook_replays (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), event_id uuid NOT NULL,
 generation integer NOT NULL, requested_by uuid NOT NULL REFERENCES app.users(id), request_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,event_id) REFERENCES app.webhook_events(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,event_id,generation)
);
-- +goose StatementBegin
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['meta_apps','wabas','phone_numbers','business_profiles','asset_sync_runs','webhook_events','webhook_facts','webhook_processing_attempts','webhook_replays'] LOOP
  EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE app.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY tenant_scope ON app.%I TO waba_runtime,waba_identity USING (organization_id=nullif(current_setting(''app.organization_id'',true),'''')::uuid) WITH CHECK (organization_id=nullif(current_setting(''app.organization_id'',true),'''')::uuid)',t);
  EXECUTE format('GRANT SELECT,INSERT ON app.%I TO waba_runtime,waba_identity',t);
  IF t NOT IN ('webhook_facts','webhook_processing_attempts','webhook_replays') THEN
   EXECUTE format('GRANT UPDATE ON app.%I TO waba_runtime,waba_identity',t);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
-- A lookup by the configured external App ID returns routing identifiers only.
GRANT SELECT ON app.meta_apps TO waba_authorizer;
CREATE POLICY meta_router ON app.meta_apps TO waba_authorizer USING(true) WITH CHECK(false);
-- +goose StatementBegin
CREATE FUNCTION app.meta_binding(p_app text) RETURNS TABLE(organization_id uuid,id uuid,callback_key uuid) LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,app AS $$
 SELECT organization_id,id,callback_key FROM app.meta_apps WHERE external_id=p_app
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.meta_binding(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.meta_binding(text) TO waba_runtime,waba_identity;
GRANT CREATE ON SCHEMA app TO waba_authorizer;
ALTER FUNCTION app.meta_binding(text) OWNER TO waba_authorizer;
REVOKE CREATE ON SCHEMA app FROM waba_authorizer;
-- Original evidence cannot be rewritten; expired ciphertext may only be purged.
-- +goose StatementBegin
CREATE FUNCTION app.protect_webhook_original() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF NEW.id<>OLD.id OR NEW.organization_id<>OLD.organization_id OR NEW.app_id<>OLD.app_id OR NEW.raw_digest<>OLD.raw_digest OR NEW.received_at<>OLD.received_at OR NEW.request_id<>OLD.request_id OR NEW.retain_until<>OLD.retain_until THEN RAISE EXCEPTION 'Original webhook is immutable'; END IF;
 IF NEW.raw_ciphertext IS DISTINCT FROM OLD.raw_ciphertext AND NOT(NEW.raw_ciphertext IS NULL AND OLD.retain_until<=now()) THEN RAISE EXCEPTION 'Original ciphertext is immutable'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_webhook BEFORE UPDATE ON app.webhook_events FOR EACH ROW EXECUTE FUNCTION app.protect_webhook_original();
REVOKE ALL ON FUNCTION app.protect_webhook_original() FROM PUBLIC;
INSERT INTO app.permissions(key,sensitive) VALUES ('meta.view',false),('meta.manage',true),('webhooks.view',false),('webhooks.payload.view',true),('webhooks.replay',true);
-- Existing roles with owners.manage receive new capabilities without role-name authorization.
-- The migrator owns these tables but FORCE RLS is temporarily lifted only in this migration transaction.
ALTER TABLE app.role_permissions NO FORCE ROW LEVEL SECURITY;
INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key)
 SELECT gen_random_uuid(),r.organization_id,r.role_id,p.key FROM app.role_permissions r CROSS JOIN app.permissions p
 WHERE r.permission_key='owners.manage' AND p.key IN ('meta.view','meta.manage','webhooks.view','webhooks.payload.view','webhooks.replay') ON CONFLICT DO NOTHING;
ALTER TABLE app.role_permissions FORCE ROW LEVEL SECURITY;
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Forward repair required'; END $$;
-- +goose StatementEnd
