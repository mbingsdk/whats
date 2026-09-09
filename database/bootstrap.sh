#!/bin/sh
set -eu
# Local disposable database only. Values come from ignored generated env_file.
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<'SQL'
\getenv migration_password MIGRATION_PASSWORD
\getenv runtime_password RUNTIME_PASSWORD
CREATE ROLE waba_authorizer NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
CREATE ROLE waba_migrator LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD :'migration_password';
CREATE ROLE waba_runtime LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD :'runtime_password';
GRANT waba_authorizer TO waba_migrator WITH INHERIT FALSE, SET TRUE;
REVOKE ALL ON DATABASE waba FROM PUBLIC;
GRANT CONNECT ON DATABASE waba TO waba_migrator, waba_runtime;
GRANT CREATE ON DATABASE waba TO waba_migrator;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE, CREATE ON SCHEMA public TO waba_migrator;
GRANT USAGE ON SCHEMA public TO waba_runtime;
SQL
