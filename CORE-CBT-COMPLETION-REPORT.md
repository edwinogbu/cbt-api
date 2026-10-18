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

## 14. Follow-up session: manual grading, LAN conflict resolution, genuine offline verification

Continuing directly from §12's priority list (manual grading → LAN conflict
resolution → genuine offline-reload/offline-exam tests, in that order).
`cbt-api` HEAD at session start: `123f853`; `elias-cbt-portal` HEAD:
`604730a`. All five previously-cited fixes (grading integrity, MarkReview
time-limit, student-route middleware, JWT security tests, PWA proxy
allowlist) were re-confirmed present before any new work started.

### 14a. Manual/essay grading - implemented, fully live-verified

Implemented using the **existing** `ExamAttempt`/`StudentAnswer`/`Result`
models - no second grading system. Changes:

- `models.StudentAnswer`: added `MarksAwarded *int`, `MarkedBy *string`,
  `MarkedAt *time.Time`, `MarkerFeedback string` (nil `MarksAwarded` means
  "not yet marked" - distinct from a legitimate 0-mark score).
- `models.Result`: added `FullyGraded bool` (`default:true`, so every
  pre-existing, necessarily-all-objective row backfills correctly).
- New `AttemptStatusPendingManualMarking` status value. Chosen over a
  boolean flag specifically so every pre-existing "completed-only" query
  (dashboards, class results, rankings) excludes a not-yet-fully-marked
  attempt with **zero** changes to those call sites.
- `ExamService.gradeAnswer()`: one shared helper now used by `AutoSave`,
  `SaveAnswer`, and `BulkSaveAnswers` alike. Found and fixed a genuine
  pre-existing bug while unifying this: `AutoSave` never set `IsCorrect`
  at all.
- New `GetPendingManualMarking`/`MarkDescriptiveAnswer` service methods,
  `GET /exams/marking/pending` and `POST /exams/marking/grade` routes
  (teacher/admin/super_admin, plus a class-ownership check mirroring the
  existing `authorizeQuestionWrite` pattern - a teacher cannot mark another
  teacher's class).
- `SubmitExam` rewritten to branch per question: objective questions score
  immediately as before; an essay question with no rubric-independent
  auto-score sets `needsManualMarking`, and the attempt's final status
  becomes `pending_manual_marking` (not `completed`) when any essay answer
  is still unmarked.
- `finalizeMarkingForAttempt`: recomputes total score/percentage/grade/
  passed from every `StudentAnswer` row (same formula as the earlier
  grading-integrity fix) once a marker submits a score, and flips the
  attempt to `completed` once nothing remains unmarked.

**Live-verified end to end** against the real API/Postgres (not just unit
tests): created an exam with 4 objective + 1 essay question → student
submitted → attempt correctly `pending_manual_marking` with a provisional
score, correctly absent from the dashboard's completed list → a teacher
saw it in `/marking/pending` with the full essay text → an unauthorized
teacher and a student were both correctly rejected (403) → an out-of-range
score was correctly rejected (400) → the authorized teacher marked it →
attempt correctly became `completed`, combined score 16/18 = 88.89%,
correctly now visible on the dashboard and cleared from the pending list.
7 new regression tests cover invalid scores, wrong-teacher rejection,
student rejection, admin bypass of class ownership, and idempotent
re-marking. Committed as `fc75fb9`.

### 14b. LAN sync conflict detection - implemented, unit-verified

§8's disclosed gap ("conflict handling is unconditional last-write-wins...
no detection") is now closed with **status-rank-based** conflict
detection, deliberately **not** timestamp-based - a School Node's clock
cannot be trusted to agree with the cloud server's, so timestamps alone
would be exactly the unreliable signal the directive warned against.

- `attemptStatusRank`: a fixed lifecycle order (`in_progress` <
  `submitted` < `pending_manual_marking` < `completed`/`graded`). An
  incoming session push is rejected if it would move the stored attempt
  *backward* in this ranking, regardless of what timestamp it carries -
  this is what makes a final submission impossible for a delayed/
  out-of-order sync to silently revert.
- Answer pushes: a genuine conflict (the incoming value differs from what's
  stored AND the parent attempt has already moved past `in_progress`) is
  rejected and logged; a brand-new answer or an identical retry is always
  accepted, so idempotent retries keep working. (The first implementation
  checked attempt-status before checking whether the answer already
  existed and differed, which rejected every legitimate first-time answer
  pushed alongside an already-finalized attempt in the same batch - caught
  and fixed before any test or commit.)
- Result pushes: rejected if the stored result is already `FullyGraded`
  and the incoming values differ; an identical retry still succeeds.
  `ResultPush`'s DTO gained a `FullyGraded` field - its absence would have
  silently zeroed this flag on the cloud side on every push, corrupting
  exactly the field manual grading (§14a) depends on.
- Every rejection is logged (`zap.Warn`) with the record type, ID, and the
  specific reason, satisfying "define how conflicts are resolved and
  logged."

6 new tests (session-downgrade-rejected, session-advance-accepted,
answer-conflict-rejected, answer-identical-retry-accepted,
result-overwrite-rejected, result-identical-retry-accepted) all pass
against a real SQLite-backed `NodeSyncService`, not mocks. Full
`go build`/`go vet`/`go test ./...`/`go test -race` all pass. **Not**
additionally re-verified at the live HTTP layer (`POST /nodesync/push`)
this session - that would need seeding a school-node API key plus
attempt/answer/result rows through several more entity layers, and the
service-layer tests already exercise the real conflict logic against a
real database, not a mock. Committed and pushed as `83dcdfd`.

### 14c. Offline PWA reload - two root causes found and FIXED; genuine offline reload now verified working

The prior session's open question (§6) was whether the "zero responses
served offline" result was a DevTools-Protocol-emulation artifact or a
real defect. This session built a **real, non-emulated** offline test
specifically to answer that: a Linux network namespace (`offlinetest`)
connected to the root namespace by a veth pair, with `socat` relays so the
production build's hardcoded `localhost:3000`/`localhost:8080` URLs work
unmodified. "Going offline" is `ip link set veth-off down` - an actual
kernel-level link-down, the same primitive used when a real network
adapter is disabled, not `context.set_offline()` or
`Network.emulateNetworkConditions`.

**Initial result: the same failure reproduces with a genuine disconnect.**
With the service worker confirmed `activated` and `controller: true`, and
the build-time precache populated (257 entries), reloading a live,
previously-visited exam-attempt page while genuinely offline produced
**zero** response/request events of any kind within 20+ seconds and the
page never rendered. This ruled out "it's just a Playwright/DevTools
quirk" as the explanation.

**Root cause #1 (the page's own response was never cacheable)**: the
exam-attempt page (`/dashboard/student/exams/[examId]/attempt/[attemptId]`)
was served with `Cache-Control: private, no-cache, no-store, max-age=0,
must-revalidate` (confirmed via the actual response header on a real
navigation) - Next.js App Router's default for a route with dynamic
segments and no `generateStaticParams`.

**Fix #1**: split the 17,800+ line client component into a thin **server**
`page.tsx` (new file, ~35 lines) that renders the unchanged original code
as a new `AttemptClient.tsx` ('use client') - Next.js disallows route
segment config exports in a file with `'use client'`, so the config has
to live in a separate server-component wrapper. `page.tsx` carries:
```ts
export async function generateStaticParams() { return []; }
export const revalidate = 3600;
```
`generateStaticParams` returning `[]` turned out to be required, not
optional: `revalidate` alone (without it) had **no effect at all** on the
response header - confirmed live by comparing headers against a
known-static sibling route (`/dashboard/student`) under the identical auth
middleware, which got a normal cacheable header while the attempt page
stayed `no-store` even with `revalidate` set. Next.js's App Router only
applies ISR-style caching to a dynamic-segment route that also calls
`generateStaticParams` (even with an empty array - there's nothing to
enumerate ahead of time); without it, the route never enters the
prerender/ISR manifest and `revalidate` is silently ignored. With both
together, the live response header changed to
`s-maxage=3600, stale-while-revalidate=31532400` and the build's route
table now marks this exact route `●` (SSG via `generateStaticParams`)
instead of `ƒ` (fully dynamic). A diff of `AttemptClient.tsx` against the
original `page.tsx` shows **exactly two differences**: the file-header
comment and the function's name (`ExamAttemptPage` → `AttemptClient`) -
zero logic changes.

**Root cause #2 (found only after fixing #1, and the deeper one): the
service worker's runtime-caching routes never matched ANYTHING, for ANY
page, ever**. Even with the attempt page's own response now cacheable,
re-running the genuine offline test still failed identically - `pages-
cache` never existed in Cache Storage at all (confirmed: `caches.keys()`
after reloading `/dashboard/student`, a plain static page, online,
multiple times, with the SW active and controlling, showed only the
build-time precache - no `pages-cache`, no `api-cache`). A manual
`caches.open('x').put(url, response)` of the exact same response
succeeded instantly, ruling out any Cache-API-level rejection. Root cause,
confirmed by reading `node_modules/workbox-routing/RegExpRoute.js`
directly: Workbox tests a `RegExp` `urlPattern` against the request's
**full `url.href`** (e.g. `"http://host:3000/dashboard/..."`), not just
the pathname. `next.config.ts`'s `runtimeCaching` patterns were all
written as `/^\/api\/.../ ` and `/^\/.*/ ` - anchored on a literal leading
`/`, which `url.href` never starts with (it always starts with
`"http://"` or `"https://"`). These patterns could not match any request,
same-origin or cross-origin, ever - meaning `api-cache` and `pages-cache`
have never actually cached anything in this PWA's entire history,
independent of anything from this session or the earlier one.

**Fix #2**: changed the three affected `urlPattern` entries in
`next.config.ts` from regexes to Workbox-style function matchers testing
`url.pathname` (what they all actually intended to test), e.g.
`({ url }) => url.pathname.startsWith("/api/verify-payment")` and the
catch-all `({ url }) => url.origin === self.location.origin`. The
project's own hand-written ambient type declaration for `next-pwa`
(`src/types/next-pwa.d.ts`) only allowed `RegExp` for `urlPattern` (an
incomplete declaration, not a real constraint - `next-pwa` ships no
types of its own, and the underlying `workbox-build` library it wraps
genuinely supports function matchers) - widened to
`RegExp | ((options: { url: URL; request?: Request }) => boolean)`.

**Security check on fix #2**: the `verify-payment` `NetworkOnly` rule,
which previously *also* never matched (same bug), now correctly
intercepts and bypasses caching for that path - fixing the catch-all
without this would have been a regression (payment verification could
have started being cached). The `api-cache` rule only ever applies to
`GET` requests (`registerRoute(..., "GET")`, confirmed in the generated
`sw.js`); the Next.js-side AI endpoints it covers
(`extract-text`/`gemini-*`) are POST-only, so the Cache API still never
actually stores a response for them - unchanged from before. The
`pages-cache` catch-all only matches same-origin requests
(`url.origin === self.location.origin`); the real Go backend API is a
different origin and was never reachable by this rule either before or
after. The cached exam-attempt page shell contains no embedded
per-student data (it's a pure client component - page.tsx passes it zero
props, and every real value is fetched/read client-side after hydration),
and Cache Storage is scoped per browser profile, never shared across
students' devices - so caching this shell introduces no cross-student
exposure.

**Genuine offline reload re-tested and PASSED**, end to end, with the
same real kernel-level-disconnect methodology, after both fixes:
- `pages-cache` now populates (confirmed non-empty for the exact attempt
  URL after an online reload).
- Reload while genuinely offline: **no error, no timeout** - 23 responses
  observed, every single one with `from_service_worker: true`, including
  the HTML document itself (status 200) and all its JS/CSS/font/icon
  dependencies.
- The page did not just return a 200 - it **rendered the real exam UI**:
  confirmed the body text progressed from a brief "Loading your exam...
  Preparing questions" state into the actual question content
  ("Question 1 of 5", "Lifecycle test Q1: 2+2=?", the countdown timer,
  "0/5" progress) within ~5 seconds, entirely offline.
- `IndexedDB` databases (`CBTDatabase`, `ExamOfflineDB`) remained present
  and queryable throughout.

**UI regression check after the fix (clicking a real answer option,
not a raw API call)**: confirmed the split did not break answer-saving.
Clicking an `OptionButton` wrote a real, encrypted `answers` record to
Dexie (`syncStatus: "PENDING"`) and queued a sync-queue entry - this app
is deliberately local-first (`saveAnswer`'s own comment: "SAVE → LOCK →
QUEUE → SYNC, never SEND → WAIT → HOPE"), so no immediate network call is
expected or correct; the durable local write is the thing that matters,
and it was confirmed present via a direct read of the raw IndexedDB
object stores, not just the UI's own summary counters (which are bound to
server-fetched `attemptData.answered_count`, not the optimistic local
state, and so don't move until the next full refetch - this is
pre-existing behavior, unchanged by the split).

**Separately confirmed still true**: the app also survives connectivity
loss fine *without* a reload, as already shown in the prior session and
re-confirmed this session (§14d) - that was never the broken path; the
reload-while-offline path specifically is what §14c fixes.

### 14d. Complete offline exam lifecycle, public internet genuinely disabled - verified

Using the same real (non-LAN-fallback, kernel-level-disconnect) namespace
as §14c, without ever reloading the page (since §14c establishes that path
is currently broken): logged in and started an exam online; answered one
question online; brought the network namespace's interface **down**
(confirmed via a raw probe through the same relay returning nothing -
genuine disconnect, not "LAN still reachable"); attempted to save a second
answer and attempted to submit, both while genuinely offline - both failed
cleanly with a client-side abort/network error, **no corrupted or partial
write reached the server** (confirmed: the attempt's `answered_count`
stayed at exactly 1, matching only what had actually reached the server
before going offline); the already-loaded page's `document.readyState`
stayed `complete` throughout (no crash) and its local IndexedDB databases
(`CBTDatabase`, `ExamOfflineDB`) remained present; reconnected (interface
back up, confirmed via a real health-check response); submitted for real
online - correct grading (2 of 5 questions answered correctly, 10/25 =
40%, grade E, matching the known-correct answers); submitting the same
already-completed attempt a second time returned the **identical** stored
result (same `result_id`, score, `submitted_at`) rather than re-grading or
erroring, confirming the existing duplicate-submission protection holds
under this exact offline-then-reconnect sequence.

**Not covered by this pass**: the app's own internal Dexie/IndexedDB sync
queue (`saveAnswerWithTransaction` + the sync manager that drains it on
reconnect) was not exercised end-to-end - this test called the real API
directly from the page's JS context to get a clean, unambiguous signal
about server-side data integrity under genuine disconnection, rather than
going through the UI's click path. Whether an answer saved through the
*actual UI* while offline is queued and auto-synced on reconnect (as
opposed to simply failing cleanly, which is what was proven) remains
unverified by this specific test, though it was already shown working in
the prior session via Playwright's `context.set_offline()` with the tab
kept open (§3e of `OFFLINE-FIRST-COMPLETION-REPORT.md`) - not re-verified
here with a genuine kernel-level disconnect specifically.

### 14e. Regression gates re-run this session

| Suite | Result |
|---|---|
| `cbt-api`: `go build ./...` | pass |
| `cbt-api`: `go vet ./...` | pass, no findings |
| `cbt-api`: `go test ./...` | pass (all packages, including 7 new manual-marking tests and 6 new nodesync conflict tests) |
| `cbt-api`: `go test -race ./internal/cbt/service/... ./internal/sync/... ./internal/nodesync/...` | pass, no races |
| `cbt-api`: JWT signing-secret fix | re-checked - still reads `JWT_SECRET` with the same production-startup refusal in both `config.go` and the actual signing path `pkg/utils/utils.go`; no hardcoded fallback has returned |
| `elias-cbt-portal`: `npx tsc --noEmit` | 0 errors (after the offline-reload fix) |
| `elias-cbt-portal`: `npm run build` (webpack, PWA gen) | pass; `/dashboard/student/exams/[examId]/attempt/[attemptId]` now shows `●` (SSG) instead of `ƒ` (dynamic) in the route table |
| `elias-cbt-portal`: `npx vitest run` | 28/28 pass |
| `elias-cbt-portal`: genuine kernel-level-disconnect offline reload | **pass** - see §14c |

### 14f. Updated priority list

1. Exercise the app's own Dexie-queue auto-sync path's *network delivery*
   specifically under a genuine kernel-level disconnect (§14d's one
   explicitly-undone item) - the local write side of this is now confirmed
   (§14c's UI regression check), only the "does it actually reach the
   server after reconnect" half via the real queue (not a raw API call)
   remains unverified with this specific disconnect method.
2. Live HTTP-layer verification of the nodesync conflict-detection logic
   (§14b) against the real running API, beyond the service-layer tests.
3. Everything still outstanding from §12 that this session did not touch:
   duplicate-question detection beyond exact-text matching, Electron code
   signing, dead-code deletion, broader LAN hardware testing.

### 14g. What this session does and does not claim

- **Does** claim offline PWA reload now works, for the exam-attempt page
  specifically: proven via a genuine kernel-level network disconnect (not
  DevTools emulation), with the service worker demonstrably serving the
  HTML document and all its dependencies from Cache Storage, and the page
  rendering the real exam UI (not a blank or error page) while offline.
  Two independent, previously-undiscovered root causes were found and
  fixed to get here (§14c) - not assumed from a cacheable header or a
  successful build alone.
- Does not claim every page in the app has been individually re-tested
  for offline reload - only the exam-attempt page was driven through the
  full genuine-disconnect test; the `pages-cache`/`api-cache` matcher fix
  is app-wide (it was broken for every same-origin page, not just this
  one), but only this one page's reload was end-to-end re-verified.
- Does not claim the LAN conflict-detection logic has been exercised
  through the live HTTP API, only through real-database service-layer
  tests.
- Does not claim the app's internal offline-sync-queue *network delivery*
  path was re-verified under a genuine (non-CDP-emulated) disconnect -
  only that the queue's local write succeeds via the real UI (confirmed
  this session) and that raw, direct API calls fail safely when offline;
  the queue's own reconnect-and-flush behavior was shown working under
  CDP-level offline emulation in the prior session, not re-verified here
  with a genuine disconnect specifically.
- Manual/essay grading (§14a), the offline PWA reload fix (§14c), and the
  exam-submission/duplicate-protection behavior under genuine
  offline-then-reconnect (§14d) are backed by full live, non-simulated
  verification end to end this session.
