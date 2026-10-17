-- Migration 0001: one Result row per ExamAttempt
--
-- Context: Phase 1 offline-strategy audit found that SubmitExam's
-- "already submitted?" check happened before any transaction was open,
-- so two concurrent submit requests for the same attempt (a client
-- retry racing the original, or a genuine duplicate) could both pass
-- the check and both create a Result row. The application-level fix
-- (row-locking the attempt inside a transaction, idempotent re-submit
-- returning the existing result) is already deployed and is the
-- primary defense - see internal/cbt/service/exam_service.go's
-- SubmitExam. This migration adds a database-level backstop so the
-- invariant holds even if application logic is ever bypassed or has a
-- bug (a direct DB write, a different code path, future refactor, etc).
--
-- NOT wired into GORM's AutoMigrate (the `results.attempt_id` column
-- intentionally still carries only a plain `index` tag, not
-- `uniqueIndex`, in internal/models/cbt.go) and must be applied by hand
-- for one specific reason: this session has no way to connect to the
-- production database to check whether duplicate Result rows already
-- exist there from before this fix. If they do, this ALTER would fail -
-- and because AUTO_MIGRATE failures call log.Fatal in cmd/server/main.go,
-- an unconditional AutoMigrate attempt could crash the server on its next
-- deploy. Applying this by hand, after running Step 1 below, avoids that
-- risk entirely.
--
-- ============================================================
-- STEP 1 - RUN THIS FIRST. Read-only, makes no changes.
-- ============================================================
-- Lists every attempt_id with more than one Result row. If this
-- returns zero rows, skip straight to Step 2. If it returns any rows,
-- STOP and resolve them before proceeding - this migration does not
-- delete or merge data automatically (per the Phase 1 instructions:
-- never delete or overwrite existing examination results without
-- explicit review). Decide, per duplicate attempt_id, which Result row
-- is authoritative (likely the earliest by created_at) and what to do
-- with the other(s) - keep both under a note, or delete the spurious
-- one only after confirming it really is a duplicate of the same
-- attempt rather than two different things that happen to share an id
-- due to some other bug.

SELECT attempt_id, COUNT(*) AS result_count, array_agg(id) AS result_ids
FROM results
WHERE deleted_at IS NULL
GROUP BY attempt_id
HAVING COUNT(*) > 1;

-- ============================================================
-- STEP 2 - only after Step 1 returns zero rows (or after you've
-- resolved every duplicate it found).
-- ============================================================
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_results_attempt_id_unique
    ON results (attempt_id)
    WHERE deleted_at IS NULL;

-- CONCURRENTLY avoids holding a write lock on the results table for the
-- duration of the index build - safe to run against a live database
-- without blocking other reads/writes, at the cost of not being
-- wrappable in the same transaction as Step 1 (Postgres disallows
-- CREATE INDEX CONCURRENTLY inside a transaction block - run Step 1
-- and Step 2 as separate statements/sessions, not as one script).
--
-- RUNNER COMPATIBILITY: this repo has no automated migration runner
-- (golang-migrate/goose/atlas) - the only automated path is GORM's
-- AutoMigrate, which this file deliberately bypasses (see above). Run
-- this by hand with a plain `psql -f migrations/0001_....sql` (or
-- paste the statements into an interactive psql session) using a
-- role with sufficient privilege to create indexes. Do NOT run it with
-- `psql -1` / `--single-transaction`, and do NOT invoke it through any
-- tool that auto-wraps a script's statements in one transaction - that
-- produces the same "CREATE INDEX CONCURRENTLY cannot run inside a
-- transaction block" error Step 2 is written to avoid, just one layer
-- up where it's less obvious why.
--
-- If Step 2 is interrupted (connection drop, killed session) partway
-- through the index build, Postgres can leave an INVALID index behind
-- rather than cleanly rolling back (a CONCURRENTLY-specific behavior).
-- Check `SELECT indexrelid::regclass, indisvalid FROM pg_index WHERE
-- indexrelid = 'idx_results_attempt_id_unique'::regclass;` after an
-- interrupted run - if indisvalid is false, DROP INDEX
-- CONCURRENTLY idx_results_attempt_id_unique; and retry Step 2 from
-- scratch rather than assuming it's usable.
--
-- The partial WHERE deleted_at IS NULL clause matches this table's
-- existing soft-delete pattern (gorm.DeletedAt) - a soft-deleted
-- Result should not block a legitimate new one for the same attempt
-- (e.g. an admin voiding a bad result and allowing a re-grade).

-- ============================================================
-- ROLLBACK
-- ============================================================
-- Same transaction-block restriction as Step 2 applies here - run this
-- on its own, not inside `psql -1` or an auto-transactional tool.
-- DROP INDEX CONCURRENTLY IF EXISTS idx_results_attempt_id_unique;
