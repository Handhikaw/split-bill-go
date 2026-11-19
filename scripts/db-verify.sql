-- SplitKita database verification.
-- Usage:
--   psql -U <user> -d <db> -f db-verify.sql
--   docker exec -i <container> psql -U <user> -d <db> -f - < db-verify.sql
--
-- Read-only. Safe to run against production.

\pset pager off

\echo '== Connection =='
SELECT
    current_database() AS database,
    current_user       AS db_user,
    version()          AS server_version;

\echo '== Row counts (all public tables) =='
SELECT
    relname AS table_name,
    (
        xpath(
            '/row/cnt/text()',
            query_to_xml(
                format('SELECT count(*) AS cnt FROM %I.%I', schemaname, relname),
                false, true, ''
            )
        )
    )[1]::text::bigint AS row_count
FROM pg_stat_user_tables
ORDER BY relname;

\echo '== Table count =='
SELECT count(*) AS public_tables
FROM pg_stat_user_tables;

\echo '== Migration objects (0001-0003) =='
SELECT item, present
FROM (
    SELECT 'bill.kind' AS item,
           EXISTS (
               SELECT 1 FROM information_schema.columns
               WHERE table_schema = 'public' AND table_name = 'bill' AND column_name = 'kind'
           ) AS present
    UNION ALL
    SELECT 'goal_bill',
           to_regclass('public.goal_bill') IS NOT NULL
    UNION ALL
    SELECT 'contribution',
           to_regclass('public.contribution') IS NOT NULL
    UNION ALL
    SELECT 'settlement_payment',
           to_regclass('public.settlement_payment') IS NOT NULL
    UNION ALL
    SELECT 'uq_users_oauth',
           EXISTS (
               SELECT 1 FROM pg_indexes
               WHERE schemaname = 'public' AND indexname = 'uq_users_oauth'
           )
) checks
ORDER BY item;

\echo '== Ownership backfill (0003) =='
SELECT 'bill tanpa created_by' AS check_name, count(*) AS rows
FROM bill
WHERE created_by IS NULL
UNION ALL
SELECT 'participant owner tanpa user_id', count(*)
FROM participant
WHERE is_owner AND user_id IS NULL;

\echo '== Sequences =='
SELECT sequencename AS sequence_name,
       last_value,
       (last_value IS NULL) AS never_used
FROM pg_sequences
WHERE schemaname = 'public'
ORDER BY sequencename;
