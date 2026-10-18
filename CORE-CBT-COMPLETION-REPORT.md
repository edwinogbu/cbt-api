# Core CBT Platform Completion Report

Continuation session: verify the actual current state of both repos (not
trust prior reports), resolve remaining core-CBT defects, and prove
end-to-end readiness in priority order (exam lifecycle and data integrity
first, security second, offline PWA third, LAN sync fourth, AI/imports
fifth, Electron sixth). Multi-role work was explicitly deferred and not
touched this session except to confirm it still works and isn't broken.

Companion document: `elias-cbt-portal/OFFLINE-FIRST-COMPLETION-REPORT.md`
covers the PWA/offline-first work in full detail; this report summarizes
and updates that finding (§6) and covers everything else.

## 1. Actual repository state (verified, not assumed)

| Repo | Branch | HEAD | Working tree |
|---|---|---|---|
| `cbt-api` | `claude/awesome-einstein-df5zdb` | `4f388e0` | clean, pushed |
| `elias-cbt-portal` | `claude/awesome-einstein-df5zdb` | `604730a` | clean, pushed |

Both confirmed via `git status`/`git log` at session start and end; no
divergence from `origin` at any point. No commit was made to `main` or
any production/release branch. No AWS CodeBuild trigger, no production
deploy, no production database or DNS change.

This session's new commits, in order:
- `cbt-api`: `bb5c784` (MarkReview time-limit fix), `4deb7ef` (JWT
  forgery regression tests), `c7aa9d3` (exams route middleware fix),
  `4f388e0` (grading-integrity fix - the most significant one).
- `elias-cbt-portal`: `604730a` was already HEAD at session start
  (PWA precaching fix from the prior session); no new frontend commits
  were needed this session - every defect found was in `cbt-api`.

## 2. JWT signing-secret fix: re-verified secure

Re-read `pkg/utils/utils.go`'s `GenerateJWT`/`ValidateJWT` and the
`jwtSecret` initializer. Confirmed still correct: reads `JWT_SECRET` from
the environment, falls back to a known-insecure default only outside
production, and `log.Fatal`s on startup if production tries to use that
default. Traced every exam handler's `studentID` back to
`middleware.GetStudentID(c)` (JWT-context derived) - never from a
client-supplied body field - so a client cannot submit someone else's
ID to act on their behalf.

Went beyond re-reading the code: wrote 5 new empirical forgery tests
(`pkg/utils/utils_test.go`, commit `4deb7ef`) that actually attack the
implementation rather than reasoning about it:
- A token signed with the wrong secret (what an attacker without the
  real server secret would have to use) - **rejected**.
- An `alg: none` unsigned token - **rejected**.
- A token with a higher-privilege payload spliced onto a validly-signed
  token's header/signature - **rejected**.
- An expired token - **rejected**.
- A legitimately-issued token - **round-trips correctly**.

All 5 pass. The secret itself is never logged (checked every
`JWT_SECRET`/`jwtSecret` reference against `log.`/`Print`/`fmt.` call
sites - only `log.Fatal` on a missing/default value in production,
which prints no secret value). **Verdict: already correct, now has
real regression coverage proving it, not just re-asserted.**

## 3. Cross-school / cross-student isolation: one real bug found and fixed

Built real, isolated test fixtures (School B, a second student in it,
with her own `students` row and class) and tested live against the
running API with real JWTs, not mocks:

- Cross-student attempt access (`GET /student/exams/attempt/:id` for
  another student's attempt): **403 unauthorized** - correct.
- Cross-student result lookup: **404, not leaked**.
- Cross-school question-bank listing: **403, role-blocked** (students
  can't reach this endpoint at all).
- Node-sync cross-school push rejection: re-confirmed via the existing
  test suite (`TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool`)
  with a real row-count assertion, not just "no error."

**Found and fixed a real functional bug while testing this** (commit
`c7aa9d3`): `POST /api/v1/exams/{start,save-answer,bulk-answer,submit}`
returned 401 "student context not found" for **every** student,
including the legitimate owner of the attempt - not a cross-tenant
leak, the opposite: a dead end for anyone using this route group, because
it was missing `StudentContextMiddleware` (present on the parallel,
actually-used `/student/exams/*` group). The live frontend doesn't call
this group for these four endpoints (confirmed via `student.service.ts`
grep - it uses `/student/exams/answer` etc.), so this wasn't reachable
through the app's own UI, but the route is real, public, documented API
surface. Fixed by threading the already-available `studentRepo` into
`setupExamRoutes` and applying the middleware to exactly those four
routes. Re-verified live, before and after: the owner's own call went
from a hard 401 to correctly reaching the ownership/time-limit checks;
a different student's call against the same attempt correctly returned
403 once the route could actually reach that check.

The `config.go:138` comment referencing "deeper tenant-isolation
findings tracked separately" points to no specific location in the
repo. Isolation was verified sound on every concrete surface tested
this session (attempts, answers, submission, results, question-bank,
nodesync) - not exhaustive across the full API surface (hundreds of
endpoints), which no single session can claim.

## 4. Examination timing, autosave, and submission integrity

Re-audited `exam_service.go`'s `SaveAnswer`, `BulkSaveAnswers`,
`AutoSave`, `MarkReview`, `SubmitExam`, and `checkWithinTimeLimit`.

**Found and fixed** (commit `bb5c784`): `MarkReview` was the one
answer-writing path missing `checkWithinTimeLimit` (the other three
were already fixed in an earlier session phase). Since it writes to the
same `student_answers` row via `IsMarked`, a device that stayed
connected past the exam deadline could still toggle mark-for-review
flags through this one path. Fixed with the same guard the other three
already have; added 2 regression tests matching the existing test
file's pattern (accept within window, reject with zero rows written
past it).

**Re-confirmed already correct, no change needed:**
- `SubmitExam`'s row-locked transaction (`SELECT ... FOR UPDATE` inside
  a DB transaction) and idempotent re-submit (a retried call for an
  already-graded attempt returns the stored result, never creates a
  second `Result` row) - existing test
  `TestSubmitExam_RetriedAfterSuccess_ReturnsExistingResult` still
  passes; the SQLite-level true-concurrency gap this test suite already
  disclosed (no real row-locking in SQLite) remains an honest, disclosed
  limitation, not newly introduced.
- `StartExam` sets `attempt.StartTime` server-side (`time.Now()`),
  immune to client clock manipulation; `checkWithinTimeLimit` compares
  server `time.Now()` against that server-set timestamp - a client
  cannot extend its own exam time by changing its local clock.
- `calculateRemainingTime` correctly clamps to 0, never negative.

**Found and fixed, the most significant defect of this session**
(commit `4f388e0`), discovered while running a real end-to-end exam
through the live API (§5 below), not from code review alone:

- **Percentage used a stale, mismatched denominator.** `SubmitExam`
  computed `percentage = totalScore / exam.TotalMarks * 100`, where
  `exam.TotalMarks` is a separate, teacher-entered field set at
  exam-creation time with nothing keeping it in sync with the marks of
  the questions actually attached afterward via `AddQuestionsToExam`.
  Reproduced live: a real 40-question, 1-mark-each exam created with
  `total_marks=15` graded a 28-correct attempt at **186.67%** - an
  impossible percentage, silently stored in the `Result` row and
  returned as `"status": "success"` to the student. Fixed by computing
  the percentage against the actual sum of the attached questions'
  marks instead.
- **Pass-mark scale mismatch.** `passed := percentage >= exam.PassMark`
  compared a 0-100 percentage directly against `PassMark`, a raw
  marks-scale threshold (the same convention a *different*,
  already-correct line in the same file already converts properly via
  `PassMarkPercentage = PassMark/TotalMarks*100`). Any real percentage
  almost always numerically exceeds a small raw `PassMark` number, so
  this made `passed` true for nearly any attempt regardless of actual
  performance - masked precisely because it looks plausible for
  reasonable scores. The identical mismatched comparison was duplicated
  in `GetStudentDashboard`'s completed-exams list, independently
  re-deriving (and silently overriding) the correctly-stored
  `Result.Passed` value on every dashboard read. Fixed both: `SubmitExam`
  now converts `PassMark` to a percentage using the same
  actual-question-marks denominator before comparing; the dashboard
  list now trusts the already-correct stored value instead of
  recomputing it.

Added 2 regression tests with a deliberately mismatched `total_marks`
and an explicit below/above-threshold attempt, confirming: percentage
now resolves to the correct ~75.0 (not >100), and a genuinely failing
attempt (25% against a 50% threshold) is now correctly `passed: false`.
Re-verified live after the fix with a fresh real attempt: 32/40 correct
→ `percentage: 80`, `grade: "A"`, `passed: true`, consistently reflected
on the student dashboard.

`go build ./...`, `go vet ./...`, `go test ./...` all clean throughout
(22 tests in `internal/cbt/service` alone, all passing, including every
new and every pre-existing test).

## 5. Complete examination lifecycle: run live, end to end

Built real test fixtures and drove the entire flow through the actual
running Go API + Postgres (not unit tests, not mocks):

1. **School/teacher/student setup**: created School B, TeacherA (made
   the class's teacher via a DB update, since no onboarding UI step for
   this was in scope), StudentB - all via direct inserts for speed,
   matching the real schema's FK requirements.
2. **Question creation**: 40 real questions created via
   `POST /api/v1/questions/create` and `/questions/bulk` as TeacherA,
   through real validation (discovered and satisfied real required
   fields: `school_id`, `session_id`, `term_id`, `class_level_id`,
   `exam_type`, `bloom_level`, `topic` - the question-creation DTO is
   strict and correctly so).
3. **Exam creation**: via the **live** `POST /api/v1/exams/create-with-context`
   endpoint - confirmed by grep that this, not the plain
   `/exams/create`, is what the real admin UI's `CreateExamModal.tsx`
   calls. Found the plain `/exams/create` endpoint was **completely
   broken** (missing `session_id`/`term_id` fields in its DTO, while
   the DB columns are `NOT NULL` with no default - every call failed
   with a Postgres UUID-syntax error) and fixed it too (bundled into
   commit `4f388e0`'s exam_dto.go change) for consistency, even though
   it wasn't the live path.
4. **Question-count and status validation**: hit and correctly
   satisfied the real `main_exam` minimum-40-questions rule and the
   "questions must be published before attaching to an exam" rule -
   both genuine, working validations, not bugs.
5. **Publish and assign**: published the exam
   (`PUT /exams/:id {"status":"published"}`) and discovered exams are
   invisible to students until explicitly assigned
   (`POST /exams/:id/assign`) - a real `exam_assignments` join
   requirement, not a bug; confirmed the exam only appeared in the
   student's `available_exams` list after assignment.
6. **Student visibility and eligibility**: confirmed the exam appeared
   correctly for the eligible student (`is_available: true`) and was
   correctly rejected for an ineligible student in a different
   school/class (`403 "exam not assigned to your class"` on
   `StartExam`, confirmed absent from that student's dashboard query).
7. **Start, answer, autosave, recovery, submit**: started a real
   attempt; answered questions through **all three** real paths -
   single real-time `save-answer`, `auto-save`, and `bulk-answer`
   (simulating an offline-sync replay) - confirmed each path's progress
   tracking updated correctly; fetched `GetAttemptState` fresh
   (simulating a reconnect/reload after an interruption) and confirmed
   all previously-saved answers and the correct remaining time were
   intact; answered the remaining questions; submitted.
8. **Automatic marking (objective questions)**: confirmed correct
   scoring for multiple-choice questions, including catching and fixing
   the grading-integrity bug in §4 along the way - the final, re-verified
   result (32/40 correct → 80%, grade A, passed true) is now provably
   correct, not just "didn't error."
9. **Manual marking (descriptive questions)**: **not implemented
   anywhere in the codebase.** `QuestionType = "essay"` exists as a
   model constant and `StudentAnswer.EssayResponse` exists as a field,
   but there is no grading endpoint, no teacher-review workflow, and no
   way to ever set `IsCorrect`/marks for an essay answer after
   submission - it would permanently score 0 with no recovery path.
   This is a genuine missing feature, not a regression; implementing a
   full manual-grading workflow (new endpoint, DTO, re-grade
   transaction, authorization, frontend UI) is a feature addition
   beyond this session's "verify and fix defects" scope, so it was not
   built - documented here as the clearest concrete gap in core exam
   functionality.
10. **Result storage and visibility**: confirmed the `Result` row is
    created exactly once (idempotent-submit tests), with the
    now-correct percentage/grade/passed, and surfaces consistently on
    the student's own dashboard.
11. **Unauthorized access**: re-confirmed (§3) that neither a different
    student nor a different school's student can start, answer, or view
    this specific exam/attempt/result.

**This is real, live, end-to-end verification** - every step above ran
against the actual Go binary and actual Postgres database, with actual
HTTP requests and actual JSON responses inspected, not a simulated or
assumed pass.

## 6. Offline PWA: status updated from the prior session

Full detail lives in `elias-cbt-portal/OFFLINE-FIRST-COMPLETION-REPORT.md`
(already committed as of `604730a`). Summary for this report:

- **Fixed and verified this engagement** (across the prior and current
  sessions): SW registration/activation/control, precache population
  (257 entries, root cause was a middleware allowlist gap blocking two
  of the SW's own runtime files), continued offline interactivity on an
  already-open exam tab (Dexie/IndexedDB-backed, independent of the SW
  cache layer - screenshot-evidenced).
- **Investigated further this session, new finding**: used Playwright
  connected via CDP to an **independently-launched** Chrome (bypassing
  Playwright's own browser-launch wrapper) and found 25 of 37 resources
  genuinely served from the SW cache while online - proving the
  earlier "0 responses ever served from the SW" result was a
  **Playwright-launch-specific test-tooling quirk**, not an app defect
  or a sandbox-wide limitation as previously suspected.
- **Still open**: a full offline *page reload* still fails
  (`net::ERR_INTERNET_DISCONNECTED`) in every configuration tested in
  this sandbox (Playwright's own launch, a persistent context, and the
  CDP-connected independent Chrome, with both `context.set_offline()`
  and raw `Network.emulateNetworkConditions`) - the navigation request
  never reaches the SW's `fetch` handler at all when "offline" is set
  via DevTools-protocol network emulation specifically, which several
  independent signals point to being a characteristic of *how browser
  automation tools simulate offline* rather than a real-world behavior
  (a real network disconnect doesn't go through the same CDP code path).
  **This remains genuinely unverified either way** - it needs a real
  desktop browser with the network adapter actually disabled, not a
  DevTools Protocol emulation flag, to get the final verdict.

## 7. Question-bank imports and AI generation

Delegated to an independent sub-agent with instructions to verify (not
rebuild) and to actually run parsers against real constructed files
rather than only reading code. Findings:

- No Python AI service exists anywhere in either repo (re-confirmed).
- `go test ./internal/cbt/dto/...` passes, covering real parsing of
  CSV, XLSX, TXT, DOCX, RTF, legacy XLS, plus PDF-garbage-rejection and
  format-detection - actual parsing, not mocked.
- Question/import authorization is correctly anchored to
  `class.TeacherID == userID`, not to a client-supplied `school_id`.
- Duplicate detection **does exist**: exact-normalized-text matching,
  both within an uploaded file and against the existing bank. Not
  fuzzy/similarity-based - a reworded duplicate would slip through
  (documented gap, not fixed - a similarity-matching upgrade is a
  feature addition, not a fix).
- Invalid rows are never silently dropped - reported with row numbers
  in both the real-upload and preview/dry-run paths.
- Upload size is bounded (10MB handler cap, 1000-row/100-per-batch
  service cap) - no unbounded-memory risk found.
- AI generation is fully async via a Redis job queue; a worker process
  does the actual provider calls, architecturally isolating an AI
  provider outage from question-bank CRUD and exam-taking, which never
  touch the AI subsystem at request time.
- Provider failover (OpenAI → Anthropic → Gemini) confirmed in
  `providers/router.go`: tries each in order, catches errors, never
  panics; each has a 60s client timeout.
- AI-generated questions go through the identical `Create`/review path
  as manual or imported ones (same draft status, same approval gate).
- One cosmetic gap found, not fixed: the DOCX import template's own
  instructional placeholder text gets mis-parsed as a spurious question
  if a teacher uploads the unedited template.

No code changes were needed in this area - nothing broken was found.

## 8. School LAN node-sync

Delegated to an independent sub-agent with instructions to re-run the
existing tests exactly and report real pass/fail results, not summarize
vaguely. Findings:

- `go test ./internal/nodesync/... ./internal/sync/... -v` and the same
  under `-race`: **7/7 pass**, no races - names:
  `TestPush_RejectsRecordsForStudentsOutsideAuthenticatedSchool`,
  `TestAuthenticateNode_RejectsWrongKeyAndAcceptsRight`,
  `TestPush_UpsertIsIdempotent`,
  `TestSync_Idempotency_SecondCallDoesNotDuplicate`,
  `TestSync_DifferentIdempotencyKeys_BothProcessed`,
  `TestSync_OwnershipCheck_RejectsOtherStudentsAttempt`,
  `TestSync_UnknownAttempt_RejectsNotFound`.
- Node identity resolved via bcrypt-checked credential lookup, never a
  client-supplied school ID.
- School-scoped push confirmed non-trivially: the rejection test
  asserts the other school's row count is 0 in the DB, not just "no
  error."
- Idempotent upsert confirmed non-trivially: asserts a row **count** of
  exactly 1 after pushing the same session twice.
- **No resume/checkpoint mechanism exists** for an interrupted sync -
  the model is "safely retry the whole batch from scratch," which is
  safe *because of* the idempotent upsert, but is not true incremental
  resume. Reported as that honest distinction, not oversold as a
  resume feature.
- **Genuine, undisclosed-until-now gap**: conflict handling is
  unconditional last-write-wins by push arrival order - no
  version/timestamp comparison, so a record modified independently on
  both the node and the cloud would have the cloud-side change silently
  overwritten by the next node push with no detection. Flagged, not
  fixed - real conflict resolution is a feature addition beyond a
  minimal fix.
- Restart safety confirmed: no in-memory sync state anywhere; every
  write is its own auto-committed DB call, and the idempotency ledger
  is DB-persisted.

**Unit-tested, not multi-device-verified**: all of the above ran as
single-process, in-memory-SQLite tests. True two-machine/two-process
LAN integration testing remains impossible in this single-container
sandbox and was not claimed as verified.

## 9. Electron desktop

Delegated to an independent sub-agent with instructions to go beyond
re-reading the prior report and actually exercise what's runnable here.
Findings:

- `electron/main.js` loads a URL (`ELIAS_CBT_APP_URL` env override, else
  `localhost:3000` dev / a placeholder packaged-app domain) - it does
  **not** spawn the `output: "standalone"` Next.js server itself;
  nothing in the current Electron app launches that build output.
- `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true` -
  correctly configured.
- `preload.js` is intentionally empty - no IPC attack surface to audit.
- Session storage uses Electron's own Chromium profile, going through
  the identical web-app code and API calls as the browser PWA - no
  divergent auth path.
- **New this session**: actually launched the real Electron binary
  under `xvfb-run --no-sandbox` with remote debugging, confirmed via
  CDP that it opened the real login page (correct title, rendered
  content) and that its service worker registered as a second devtools
  target inside the same window - a real screenshot was captured. This
  is a stronger result than the prior session's "built a valid
  installer artifact" - this pass confirmed the **app actually runs and
  renders the real product**.
- No code defect found; no changes made.
- **Still genuinely unverifiable in this sandbox**: installing/running
  the Windows `.exe` or a macOS `.dmg` on real hardware; code signing
  (no certificate); auto-update (no publish target configured).

## 10. Automated test and regression gates - final confirmed state

| Suite | Result |
|---|---|
| `cbt-api`: `go build ./...` | pass |
| `cbt-api`: `go vet ./...` | pass, no findings |
| `cbt-api`: `go test ./...` | pass - 22 tests in `internal/cbt/service` (4 new this session), plus `internal/cbt/dto`, `internal/sync/service`, `internal/nodesync/service`, `pkg/utils` (5 new JWT tests) |
| `cbt-api`: `go test -race ./internal/sync/... ./internal/nodesync/...` | pass, no races |
| `elias-cbt-portal`: `npx tsc --noEmit` | **0 errors** (re-confirmed, not assumed) |
| `elias-cbt-portal`: `npx vitest run` | 28/28 pass |
| `elias-cbt-portal`: production build | pass (webpack, PWA generation intact) |
| Exam lifecycle (live, §5) | pass, after one critical fix |
| Auth/authz regression (live + new tests, §2-3) | pass, after one functional-bug fix |
| LAN sync (§8) | pass (unit-level) |
| Electron (§9) | runs and renders correctly in this sandbox |

## 11. Files changed this session, and why

**cbt-api:**
- `internal/cbt/service/exam_service.go` - added `checkWithinTimeLimit`
  to `MarkReview`; fixed `SubmitExam`'s percentage denominator and
  pass-mark scale mismatch; fixed the same scale mismatch in
  `GetStudentDashboard`'s completed-exams list.
- `internal/cbt/service/exam_service_test.go` - 4 new regression tests;
  widened the test DB's minimal `question_bank` table to match every
  GORM-mapped column (needed by the new code path's full-struct scan).
- `internal/cbt/dto/exam_dto.go` - added required `SessionID`/`TermID`
  to `CreateExamRequest` (fixes the always-broken plain
  `/exams/create`, not the live path but real public API surface).
- `pkg/utils/utils_test.go` (new) - 5 JWT forgery/security tests.
- `api/routes/routes.go` - applied `StudentContextMiddleware` to
  `/exams/{start,save-answer,bulk-answer,submit}` (fixes the
  always-401 bug for every student on that route group).

**elias-cbt-portal:** no changes needed this session (the one open item,
§6, needs a real browser to resolve, not more code changes based on
what's known so far).

## 12. Blockers and concrete next steps, by priority

1. Resolve the offline-reload DevTools-emulation-vs-real-disconnect
   question (§6) against a real desktop browser with the network
   adapter actually disabled.
2. Decide on and implement manual/descriptive-question grading (§5.9) -
   currently entirely unimplemented; any exam containing an essay
   question will permanently score it 0 with no teacher review path.
3. Implement real conflict detection for node-sync (§8) - currently
   unconditional last-write-wins with no detection.
4. Upgrade duplicate-question detection beyond exact-text matching (§7)
   if near-duplicate detection is wanted.
5. Everything else flagged in
   `OFFLINE-FIRST-COMPLETION-REPORT.md`'s own remaining-work list
   (precaching was already fixed; that list's other items - Electron
   code signing, dead-code deletion, broader LAN hardware testing -
   still stand).

## 13. What this session does NOT claim

- Does not claim the entire CBT platform is complete. Multiple genuine
  gaps are documented above (manual grading, LAN conflict resolution,
  offline-reload-on-a-real-browser) rather than hidden or assumed fixed.
- Does not claim true multi-device LAN or Windows-runtime verification -
  both remain sandbox-impossible and are stated as such, not inferred
  from unit tests or a build artifact.
- Does not claim the full API surface was isolation-tested - only the
  concrete surfaces actually exercised this session.
- The grading-integrity bug (§4/§5) was live in the deployed code before
  this session and is now fixed and re-verified live, with regression
  tests - this is the single most consequential finding and fix of this
  session, since it could have let *any* exam silently produce
  impossible or wrongly-passing results.
