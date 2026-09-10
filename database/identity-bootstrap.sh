#!/bin/sh
set -eu
# Idempotent local role setup; executed as the container administrator, never by API.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
\getenv identity_password IDENTITY_PASSWORD
SELECT 'CREATE ROLE waba_identity LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS'
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname='waba_identity') \gexec
ALTER ROLE waba_identity PASSWORD :'identity_password';
GRANT CONNECT ON DATABASE waba TO waba_identity;
GRANT USAGE ON SCHEMA public TO waba_identity;
SQL
