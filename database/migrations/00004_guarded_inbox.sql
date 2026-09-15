-- +goose Up
-- Sprint 3 only: Inbox domain and guarded text dispatch. No campaign schemas.
CREATE TABLE app.inbox_settings (
 organization_id uuid PRIMARY KEY REFERENCES app.organizations(id),
 sending_enabled boolean NOT NULL DEFAULT false,
 billing_currency_state text NOT NULL DEFAULT 'UNKNOWN' CHECK(billing_currency_state IN ('UNKNOWN','VERIFIED')),
 billing_currency text, currency_evidence text,
 policy_revision bigint NOT NULL DEFAULT 1,
 active_rate_publication_id uuid,
 CHECK((billing_currency_state='UNKNOWN' AND billing_currency IS NULL) OR (billing_currency_state='VERIFIED' AND billing_currency IS NOT NULL AND currency_evidence IS NOT NULL AND billing_currency ~ '^[A-Z]{3}$' AND length(currency_evidence)>10))
);
CREATE TABLE app.recipient_identities (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 phone_id uuid NOT NULL, identity_kind text NOT NULL DEFAULT 'WA_ID' CHECK(identity_kind='WA_ID'),
 identity_value text NOT NULL CHECK(identity_value ~ '^[0-9]{5,32}$'),
 display_name text NOT NULL DEFAULT '', suppressed boolean NOT NULL DEFAULT false,
 is_test boolean NOT NULL DEFAULT false, campaign_excluded boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,phone_id) REFERENCES app.phone_numbers(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,phone_id,identity_value), UNIQUE(organization_id,id,phone_id)
);
CREATE TABLE app.conversations (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 phone_id uuid NOT NULL, recipient_id uuid NOT NULL,
 assigned_team_id uuid, assigned_member_id uuid,
 status text NOT NULL DEFAULT 'OPEN' CHECK(status IN ('OPEN','RESOLVED','SNOOZED')),
 priority text NOT NULL DEFAULT 'NORMAL' CHECK(priority IN ('LOW','NORMAL','HIGH','URGENT')),
 snoozed_until timestamptz,
 handoff_state text NOT NULL DEFAULT 'WAITING_AGENT' CHECK(handoff_state IN ('WAITING_AGENT','HUMAN','BOT')),
 handoff_epoch bigint NOT NULL DEFAULT 1, assignment_revision bigint NOT NULL DEFAULT 1,
 revision bigint NOT NULL DEFAULT 1, inbound_sequence bigint NOT NULL DEFAULT 0,
 last_activity_at timestamptz NOT NULL, last_inbound_at timestamptz, last_outbound_at timestamptz,
 last_eligible_inbound_message_id uuid, window_basis_at timestamptz, window_opened_at timestamptz, window_expires_at timestamptz,
 window_policy text, window_confidence text NOT NULL DEFAULT 'UNKNOWN' CHECK(window_confidence IN ('UNKNOWN','AUTHENTICATED_TEXT')),
 free_entry_observation jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,phone_id) REFERENCES app.phone_numbers(organization_id,id),
 FOREIGN KEY(organization_id,recipient_id,phone_id) REFERENCES app.recipient_identities(organization_id,id,phone_id),
 FOREIGN KEY(organization_id,assigned_team_id) REFERENCES app.teams(organization_id,id),
 FOREIGN KEY(organization_id,assigned_member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,phone_id,recipient_id), UNIQUE(organization_id,id,phone_id)
);
CREATE INDEX conversation_inbox ON app.conversations(organization_id,assigned_team_id,last_activity_at DESC,id);
CREATE TABLE app.conversation_reads (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 conversation_id uuid NOT NULL, member_id uuid NOT NULL, inbound_watermark bigint NOT NULL DEFAULT 0 CHECK(inbound_watermark>=0),
 manual_unread boolean NOT NULL DEFAULT false, updated_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,conversation_id) REFERENCES app.conversations(organization_id,id),
 FOREIGN KEY(organization_id,member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,conversation_id,member_id)
);
CREATE TABLE app.messages (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 conversation_id uuid NOT NULL, phone_id uuid NOT NULL, provider_id text,
 direction text NOT NULL CHECK(direction IN ('INBOUND','OUTBOUND')),
 message_type text NOT NULL CHECK(message_type IN ('TEXT','IMAGE','VIDEO','AUDIO','DOCUMENT','STICKER','CONTACTS','LOCATION','REACTION','INTERACTIVE','TEMPLATE','UNKNOWN')),
 content jsonb NOT NULL DEFAULT '{}', text_body text NOT NULL DEFAULT '',
 context_provider_id text, provider_at timestamptz, received_at timestamptz NOT NULL DEFAULT now(),
 inbound_sequence bigint, source_event_id uuid,
 processing_state text NOT NULL DEFAULT 'MATERIALIZED',
 delivery_state text NOT NULL DEFAULT 'RECEIVED' CHECK(delivery_state IN ('RECEIVED','PENDING_LOCAL','ACCEPTED','SENT','DELIVERED','READ','FAILED','UNCERTAIN','UNKNOWN_CONFLICT')),
 error_code text, revision bigint NOT NULL DEFAULT 1,
 FOREIGN KEY(organization_id,conversation_id,phone_id) REFERENCES app.conversations(organization_id,id,phone_id),
 FOREIGN KEY(organization_id,source_event_id) REFERENCES app.webhook_events(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,phone_id,provider_id),
 UNIQUE(organization_id,id,conversation_id)
);
CREATE INDEX message_thread ON app.messages(organization_id,conversation_id,received_at,id);
CREATE INDEX message_search ON app.messages USING gin(to_tsvector('simple',text_body));
ALTER TABLE app.conversations ADD FOREIGN KEY(organization_id,last_eligible_inbound_message_id,id) REFERENCES app.messages(organization_id,id,conversation_id);
CREATE TABLE app.message_statuses (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), phone_id uuid NOT NULL,
 provider_id text NOT NULL, status text NOT NULL, provider_at timestamptz,
 pricing_metadata jsonb NOT NULL DEFAULT '{}', error_code integer,
 source_event_id uuid NOT NULL, fact_hash text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,phone_id) REFERENCES app.phone_numbers(organization_id,id),
 FOREIGN KEY(organization_id,source_event_id) REFERENCES app.webhook_events(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,phone_id,fact_hash)
);
CREATE INDEX status_correlation ON app.message_statuses(organization_id,phone_id,provider_id);
CREATE TABLE app.inbox_materializations (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), event_id uuid NOT NULL,
 state text NOT NULL CHECK(state IN ('MATERIALIZED','UNAVAILABLE')), error_code text,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,event_id) REFERENCES app.webhook_events(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,event_id)
);
CREATE TABLE app.internal_notes (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), conversation_id uuid NOT NULL,
 author_member_id uuid NOT NULL, body text NOT NULL CHECK(length(body)<=8000), revision bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), redacted_at timestamptz,
 FOREIGN KEY(organization_id,conversation_id) REFERENCES app.conversations(organization_id,id),
 FOREIGN KEY(organization_id,author_member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id)
);
CREATE TABLE app.note_mentions (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), note_id uuid NOT NULL, member_id uuid NOT NULL,
 FOREIGN KEY(organization_id,note_id) REFERENCES app.internal_notes(organization_id,id),
 FOREIGN KEY(organization_id,member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,note_id,member_id)
);
CREATE TABLE app.inbox_presence (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), conversation_id uuid NOT NULL,
 member_id uuid NOT NULL, expires_at timestamptz NOT NULL,
 FOREIGN KEY(organization_id,conversation_id) REFERENCES app.conversations(organization_id,id),
 FOREIGN KEY(organization_id,member_id) REFERENCES app.organization_members(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,conversation_id,member_id)
);
CREATE TABLE app.inbox_events (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), conversation_id uuid NOT NULL,
 event_type text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,conversation_id) REFERENCES app.conversations(organization_id,id), UNIQUE(organization_id,id)
);
CREATE INDEX inbox_event_replay ON app.inbox_events(organization_id,created_at,id);
CREATE TABLE app.pricing_policies (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), version text NOT NULL,
 kind text NOT NULL CHECK(kind='SERVICE_ZERO'), source_url text NOT NULL, source_sha256 text NOT NULL CHECK(length(source_sha256)=64),
 review_evidence text NOT NULL, published_at timestamptz NOT NULL,
 effective_from timestamptz NOT NULL, effective_to timestamptz NOT NULL, next_review_at timestamptz NOT NULL,
 CHECK(effective_to>effective_from), UNIQUE(organization_id,id), UNIQUE(organization_id,version)
);
CREATE TABLE app.rate_imports (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 importer_member_id uuid NOT NULL, revision bigint NOT NULL DEFAULT 1,
 state text NOT NULL DEFAULT 'DRAFT' CHECK(state IN ('DRAFT','VALIDATED','DIFFED','IN_REVIEW','CHANGES_REQUIRED','APPROVED','PUBLISHED')),
 source_url text NOT NULL, source_sha256 text NOT NULL CHECK(length(source_sha256)=64), retrieved_at timestamptz NOT NULL,
 artifact jsonb NOT NULL, import_hash text NOT NULL, correction_reason text NOT NULL DEFAULT '', validation_report jsonb, diff_report jsonb,
 base_publication_id uuid, reviewer_member_id uuid, reviewed_hash text, reviewed_at timestamptz,
 next_review_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,importer_member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,reviewer_member_id) REFERENCES app.organization_members(organization_id,id),
 CHECK(reviewer_member_id IS NULL OR reviewer_member_id<>importer_member_id), UNIQUE(organization_id,id)
);
CREATE TABLE app.rate_publications (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), import_id uuid NOT NULL,
 publisher_member_id uuid NOT NULL, reviewer_member_id uuid NOT NULL, snapshot jsonb NOT NULL, snapshot_hash text NOT NULL,
 source_sha256 text NOT NULL, next_review_at timestamptz NOT NULL, supersedes_publication_id uuid,
 published_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,import_id) REFERENCES app.rate_imports(organization_id,id),
 FOREIGN KEY(organization_id,publisher_member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,reviewer_member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,supersedes_publication_id) REFERENCES app.rate_publications(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,import_id)
);
ALTER TABLE app.rate_imports ADD FOREIGN KEY(organization_id,base_publication_id) REFERENCES app.rate_publications(organization_id,id);
ALTER TABLE app.inbox_settings ADD FOREIGN KEY(organization_id,active_rate_publication_id) REFERENCES app.rate_publications(organization_id,id);
CREATE TABLE app.pricing_authorizations (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), conversation_id uuid NOT NULL,
 actor_member_id uuid NOT NULL, session_id uuid NOT NULL REFERENCES app.sessions(id),
 token_digest text NOT NULL, scope_hash text NOT NULL, scope_snapshot jsonb NOT NULL, window_evidence jsonb NOT NULL, policy_id uuid NOT NULL,
 assignment_revision bigint NOT NULL, policy_revision bigint NOT NULL,
 expires_at timestamptz NOT NULL, consumed_intent_id uuid,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,conversation_id) REFERENCES app.conversations(organization_id,id),
 FOREIGN KEY(organization_id,actor_member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,policy_id) REFERENCES app.pricing_policies(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(token_digest)
);
CREATE TABLE app.outbound_intents (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), conversation_id uuid NOT NULL,
 actor_member_id uuid NOT NULL, session_id uuid NOT NULL REFERENCES app.sessions(id), phone_id uuid NOT NULL,
 authorization_id uuid NOT NULL, message_id uuid NOT NULL, client_key text NOT NULL CHECK(length(client_key) BETWEEN 16 AND 128),
 scope_hash text NOT NULL, text_body text NOT NULL CHECK(length(text_body) BETWEEN 1 AND 4096),
 state text NOT NULL CHECK(state IN ('QUEUED','RESERVED','DISPATCHING','ACCEPTED','BLOCKED','FAILED','UNCERTAIN')),
 error_code text, provider_id text, dispatch_started_at timestamptz, finished_at timestamptz, claim_token uuid, claim_until timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,conversation_id,phone_id) REFERENCES app.conversations(organization_id,id,phone_id),
 FOREIGN KEY(organization_id,actor_member_id) REFERENCES app.organization_members(organization_id,id),
 FOREIGN KEY(organization_id,authorization_id) REFERENCES app.pricing_authorizations(organization_id,id),
 FOREIGN KEY(organization_id,message_id,conversation_id) REFERENCES app.messages(organization_id,id,conversation_id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,actor_member_id,client_key), UNIQUE(organization_id,authorization_id)
);
ALTER TABLE app.pricing_authorizations ADD FOREIGN KEY(organization_id,consumed_intent_id) REFERENCES app.outbound_intents(organization_id,id);
CREATE TABLE app.send_attempts (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), intent_id uuid NOT NULL,
 state text NOT NULL CHECK(state IN ('DISPATCHING','ACCEPTED','REJECTED','UNCERTAIN')),
 started_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz, provider_id text, error_code text,
 FOREIGN KEY(organization_id,intent_id) REFERENCES app.outbound_intents(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,intent_id)
);
CREATE TABLE app.test_send_slots (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), phone_id uuid NOT NULL,
 recipient_id uuid NOT NULL, intent_id uuid NOT NULL,
 FOREIGN KEY(organization_id,phone_id) REFERENCES app.phone_numbers(organization_id,id),
 FOREIGN KEY(organization_id,recipient_id,phone_id) REFERENCES app.recipient_identities(organization_id,id,phone_id),
 FOREIGN KEY(organization_id,intent_id) REFERENCES app.outbound_intents(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,phone_id)
);
CREATE TABLE app.budget_periods (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id),
 name text NOT NULL, currency text NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 starts_at timestamptz NOT NULL, ends_at timestamptz NOT NULL,
 hard_limit numeric(24,8) NOT NULL CHECK(hard_limit>=0), revision bigint NOT NULL DEFAULT 1,
 CHECK(ends_at>starts_at), UNIQUE(organization_id,id)
);
CREATE TABLE app.budget_reservations (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), period_id uuid NOT NULL,
 intent_id uuid NOT NULL, amount numeric(24,8) NOT NULL CHECK(amount>0), currency text NOT NULL,
 state text NOT NULL CHECK(state IN ('RESERVED','POSTED','RELEASED','UNCERTAIN')),
 FOREIGN KEY(organization_id,period_id) REFERENCES app.budget_periods(organization_id,id),
 FOREIGN KEY(organization_id,intent_id) REFERENCES app.outbound_intents(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,period_id,intent_id)
);
CREATE TABLE app.budget_ledger (
 id uuid PRIMARY KEY, organization_id uuid NOT NULL REFERENCES app.organizations(id), reservation_id uuid NOT NULL,
 entry_kind text NOT NULL CHECK(entry_kind IN ('RESERVE','RELEASE','POST','ADJUSTMENT')),
 amount numeric(24,8) NOT NULL, evidence_kind text NOT NULL CHECK(evidence_kind IN ('RESERVED_EXPOSURE','INTERNAL_ESTIMATE','RECONCILED_INVOICE')),
 source_ref text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(organization_id,reservation_id) REFERENCES app.budget_reservations(organization_id,id),
 UNIQUE(organization_id,id), UNIQUE(organization_id,reservation_id,entry_kind,source_ref)
);

-- +goose StatementBegin
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['inbox_settings','recipient_identities','conversations','conversation_reads','messages','message_statuses','inbox_materializations','internal_notes','note_mentions','inbox_presence','inbox_events','pricing_policies','rate_imports','rate_publications','pricing_authorizations','outbound_intents','send_attempts','test_send_slots','budget_periods','budget_reservations','budget_ledger'] LOOP
  EXECUTE format('ALTER TABLE app.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE app.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY tenant_scope ON app.%I TO waba_runtime,waba_identity USING (organization_id=nullif(current_setting(''app.organization_id'',true),'''')::uuid) WITH CHECK (organization_id=nullif(current_setting(''app.organization_id'',true),'''')::uuid)',t);
  EXECUTE format('GRANT SELECT,INSERT ON app.%I TO waba_runtime,waba_identity',t);
  IF t NOT IN ('message_statuses','inbox_materializations','inbox_events','pricing_policies','rate_publications','test_send_slots','budget_ledger','note_mentions') THEN
   EXECUTE format('GRANT UPDATE ON app.%I TO waba_runtime,waba_identity',t);
  END IF;
 END LOOP;
END $$;
-- +goose StatementEnd
-- Shared permission expression; SECURITY INVOKER preserves transaction tenant scope.
-- +goose StatementBegin
CREATE FUNCTION app.inbox_permission(p_member uuid,p_key text,p_team uuid,p_subject uuid) RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,app AS $$
 SELECT EXISTS(
 SELECT FROM app.member_roles mr JOIN app.role_permissions rp ON rp.organization_id=mr.organization_id AND rp.role_id=mr.role_id
 WHERE mr.member_id=p_member AND rp.permission_key=p_key AND (mr.expires_at IS NULL OR mr.expires_at>now())
 AND (mr.scope_kind='ORG' OR (mr.scope_kind='TEAM' AND mr.team_id=p_team AND EXISTS(SELECT FROM app.teams t WHERE t.id=p_team AND t.archived_at IS NULL)) OR (mr.scope_kind='SELF' AND p_subject=p_member))
 UNION ALL SELECT FROM app.team_members tm JOIN app.teams t ON t.organization_id=tm.organization_id AND t.id=tm.team_id
 JOIN app.team_roles tr ON tr.organization_id=t.organization_id AND tr.team_id=t.id
 JOIN app.role_permissions rp ON rp.organization_id=tr.organization_id AND rp.role_id=tr.role_id
 WHERE tm.member_id=p_member AND t.id=p_team AND t.archived_at IS NULL AND rp.permission_key=p_key)
$$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.inbox_permission(uuid,text,uuid,uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION app.inbox_permission(uuid,text,uuid,uuid) TO waba_identity;
-- +goose StatementBegin
CREATE FUNCTION app.inbox_seed() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 INSERT INTO app.inbox_settings(organization_id) VALUES(NEW.id);
 INSERT INTO app.pricing_policies(id,organization_id,version,kind,source_url,source_sha256,review_evidence,published_at,effective_from,effective_to,next_review_at)
 VALUES(gen_random_uuid(),NEW.id,'META_SERVICE_2026_09_OWNER_REVIEW','SERVICE_ZERO',
 'https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing',
 'ea440b87f09dadfc89ae21449fccf51807d2ba308da1aaf81adf080b8e22cca4',
 'Owner Gate C research acceptance and zero-cost implementation authorization, 2026-09-15; no numeric rate publication',
 now(),'2026-07-01T00:00:00Z','2026-10-01T00:00:00Z','2026-09-21T00:00:00Z');
 RETURN NEW;
END $$;
-- +goose StatementEnd
REVOKE ALL ON FUNCTION app.inbox_seed() FROM PUBLIC;
CREATE TRIGGER inbox_new_organization AFTER INSERT ON app.organizations FOR EACH ROW EXECUTE FUNCTION app.inbox_seed();
-- Migration-only backfill, before restoring FORCE RLS. No runtime bypass.
ALTER TABLE app.organizations NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.inbox_settings NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.pricing_policies NO FORCE ROW LEVEL SECURITY;
INSERT INTO app.inbox_settings(organization_id) SELECT id FROM app.organizations;
INSERT INTO app.pricing_policies(id,organization_id,version,kind,source_url,source_sha256,review_evidence,published_at,effective_from,effective_to,next_review_at)
 SELECT gen_random_uuid(),id,'META_SERVICE_2026_09_OWNER_REVIEW','SERVICE_ZERO',
 'https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing',
 'ea440b87f09dadfc89ae21449fccf51807d2ba308da1aaf81adf080b8e22cca4',
 'Owner Gate C research acceptance and zero-cost implementation authorization, 2026-09-15; no numeric rate publication',
 now(),'2026-07-01T00:00:00Z','2026-10-01T00:00:00Z','2026-09-21T00:00:00Z' FROM app.organizations;
ALTER TABLE app.organizations FORCE ROW LEVEL SECURITY;
ALTER TABLE app.inbox_settings FORCE ROW LEVEL SECURITY;
ALTER TABLE app.pricing_policies FORCE ROW LEVEL SECURITY;
-- +goose StatementBegin
CREATE FUNCTION app.protect_inbox_immutable() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN RAISE EXCEPTION 'Immutable evidence; append a correction'; END $$;
-- +goose StatementEnd
-- +goose StatementBegin
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['pricing_policies','rate_publications','message_statuses','inbox_events','budget_ledger','test_send_slots'] LOOP
  EXECUTE format('CREATE TRIGGER immutable_evidence BEFORE UPDATE OR DELETE ON app.%I FOR EACH ROW EXECUTE FUNCTION app.protect_inbox_immutable()',t);
 END LOOP;
END $$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE FUNCTION app.protect_inbox_intent() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF ROW(NEW.id,NEW.organization_id,NEW.conversation_id,NEW.actor_member_id,NEW.session_id,NEW.phone_id,NEW.authorization_id,NEW.message_id,NEW.client_key,NEW.scope_hash,NEW.text_body,NEW.created_at)
 IS DISTINCT FROM ROW(OLD.id,OLD.organization_id,OLD.conversation_id,OLD.actor_member_id,OLD.session_id,OLD.phone_id,OLD.authorization_id,OLD.message_id,OLD.client_key,OLD.scope_hash,OLD.text_body,OLD.created_at)
 THEN RAISE EXCEPTION 'Immutable intent scope'; END IF;
 IF OLD.state IN ('ACCEPTED','FAILED','BLOCKED') AND NEW IS DISTINCT FROM OLD THEN RAISE EXCEPTION 'Terminal intent'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_intent BEFORE UPDATE ON app.outbound_intents FOR EACH ROW EXECUTE FUNCTION app.protect_inbox_intent();
-- +goose StatementBegin
CREATE FUNCTION app.protect_rate_import() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,app AS $$
BEGIN
 IF OLD.state='PUBLISHED' THEN RAISE EXCEPTION 'Published import immutable'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER immutable_published_import BEFORE UPDATE OR DELETE ON app.rate_imports FOR EACH ROW EXECUTE FUNCTION app.protect_rate_import();
REVOKE ALL ON FUNCTION app.protect_inbox_immutable(),app.protect_inbox_intent(),app.protect_rate_import() FROM PUBLIC;
INSERT INTO app.permissions(key,sensitive) VALUES
 ('inbox.view',false),('messages.send',false),('conversations.manage',false),('conversations.assign',false),
 ('notes.write',false),('notes.redact',true),('pricing.view',false),('pricing.registry.import',true),
 ('pricing.registry.review',true),('pricing.registry.publish',true),('budgets.manage',true),('sending.manage',true);
ALTER TABLE app.role_permissions NO FORCE ROW LEVEL SECURITY;
INSERT INTO app.role_permissions(id,organization_id,role_id,permission_key)
 SELECT gen_random_uuid(),r.organization_id,r.role_id,p.key FROM app.role_permissions r CROSS JOIN app.permissions p
 WHERE r.permission_key='owners.manage' AND p.key IN ('inbox.view','messages.send','conversations.manage','conversations.assign','notes.write','notes.redact','pricing.view','pricing.registry.import','pricing.registry.review','pricing.registry.publish','budgets.manage','sending.manage') ON CONFLICT DO NOTHING;
ALTER TABLE app.role_permissions FORCE ROW LEVEL SECURITY;
GRANT DELETE ON app.inbox_presence TO waba_runtime,waba_identity;
CREATE INDEX inbox_presence_expiry ON app.inbox_presence(expires_at);
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Forward repair required'; END $$;
-- +goose StatementEnd
