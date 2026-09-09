-- Read-only operator query. No SQL text, parameters or customer content.
SELECT clock_timestamp() AS sampled_at,
       pid,
       wait_event_type,
       wait_event,
       clock_timestamp() - xact_start AS transaction_elapsed,
       clock_timestamp() - query_start AS query_elapsed,
       pg_blocking_pids(pid) AS blocking_pids
FROM pg_stat_activity
WHERE datname = current_database()
  AND pid <> pg_backend_pid()
  AND backend_type = 'client backend'
ORDER BY pid;
