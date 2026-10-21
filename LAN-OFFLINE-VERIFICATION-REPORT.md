# LAN/Offline-First Verification Report — Round 2

This report covers the work done in direct response to the most recent,
most specific instruction in this engagement: implement and **actually
verify**, not just build, a complete offline-first, LAN-only CBT system,
prioritized ahead of any new cloud-only feature, with honest disclosure of
what was tested against a real disconnected network versus simulated.

It supersedes nothing in `OFFLINE-FIRST-COMPLETION-REPORT.md` or
`MASTER-PLAN-COMPLETION-REPORT.md` — both predate this round and cover
different work (the TypeScript backlog, JWT fix, multi-role switching, the
PWA registration/precaching bugs, Platform Owner tenant registry). This
report now covers six rounds of work: the five initial tasks (§1-§4),
a round (§5) that ran the remaining items from your Section 8
end-to-end checklist, the health-monitoring dashboard (noted inline in
§4 item 5), a round (§6) that re-verified CSV/Excel/Word/TXT bulk
question import against the real HTTP API, a round (§7) that closed
out the remaining JSON/RTF/PDF/XLS formats the same way, and a round
(§8) that re-verified report printing - finding and fixing a real
crash and a complete absence of print styling.

## 1. Repositories, branches, commits — all pushed, nothing uncommitted

| Repo | Branch | Working tree at report time |
|---|---|---|
| `cbt-api` | `claude/awesome-einstein-df5zdb` | clean, all commits pushed |
| `elias-cbt-portal` | `claude/awesome-einstein-df5zdb` | clean, all commits pushed |

Commits produced by this round of work, in order:

| Commit | Repo | What |
|---|---|---|
| `5afdb10` | cbt-api | **Fix auto-grading: every answer to an options+correct_option_keys question scored wrong** — found via a real end-to-end exam cycle, not code review |
| `476e088` | cbt-api | Database backup/restore tooling for School CBT Nodes |
| `d7f155b` | cbt-api | Native Windows service wrapper for the School CBT Node LAN server |
| `26873f7` | elias-cbt-portal | Fix real cross-user/tenant data-leak: service worker cached backend API responses |
| `6c1f141` | elias-cbt-portal | electron-builder GitHub Releases publish config (infrastructure only, unused — installer publish was explicitly paused per your instruction to prioritize offline/LAN work) |
| `ae2853c` | elias-cbt-portal | Runtime server-address configuration for the Electron desktop app |
| `30e459c` | cbt-api | **Round 3** — Fix `StartExam`: stored the wrong ID type, breaking every subsequent save/submit on attempts started through that route — see §5 |
| `512b839` | cbt-api | `GET /admin/system/health` — backend for the node health-monitoring dashboard |
| `df40426` | elias-cbt-portal | Node Health dashboard page (`/dashboard/admin/node-health`) |
| `7e73c32` | cbt-api | **Round 4** — Fix bulk-import: essay questions failed in every format (CSV/Excel/Word/TXT), every time — see §6 |
| `ed18af8` | elias-cbt-portal | **Round 6** — Fix a real crash blocking every exam result page (snake_case/camelCase mismatch) — see §8a |
| `3788592` | elias-cbt-portal | **Round 6** — Add a real print stylesheet; printing produced the full dashboard chrome, not a report — see §8b |

Two earlier commits this session (`ac0ff71`, `899aa3a` in cbt-api) were
Platform Owner / tenant-branding backend work queued before your
"work offline first" redirection arrived; they're unrelated to this
report's scope and already covered by prior reporting.

## 2. What was actually tested, and how — task by task

### Task 1 — Full offline exam cycle on an isolated LAN node

**Setup**: stood up a second, fully independent instance of the `cbt-api`
Go binary — its own Postgres database, its own Redis, its own port
(8081), its own JWT secret — modeling a real School CBT Node rather than
reusing the main dev stack. This matters because it's the only way a bug
specific to the LAN-node code path (as opposed to the cloud path) would
surface.

**What was driven through its real HTTP API, not asserted from reading
code**: teacher login → question creation (had to fix two real blockers
along the way: exam-type minimum-question counts, and a class's
`teacher_id` not being set — both environment/seed-data issues, not bugs)
→ bulk-publish questions → create an exam → enroll a student → student
login → start exam → submit answers → **auto-grading** → view result.

**What this found**: a real, previously-undetected bug.
`ExamService.gradeAnswer` only compared a submitted answer against the
legacy single-string `CorrectAnswer` field. Every question authored using
the modern `options` + `correct_option_keys` format — the format the
question-bank UI actually produces — was graded wrong on **every**
submission, regardless of what the student picked. This is the kind of
defect that is invisible from reading the happy-path code and only
surfaces by actually running a question through grading. Fixed, and a
regression test (`TestGradeAnswer_UsesCorrectOptionKeys`) now locks in the
case that failed.

**Verification method for the fix**: the established red/green
discipline — reverted the fix, confirmed the new test failed for the
stated reason, restored it, confirmed green. Also caught and fixed a
second, related bug along the way: `CorrectOptionKeys.Scan()` only
handled the `[]byte` form Postgres's jsonb driver returns, not the
`string` form SQLite's test driver returns — widened to handle both.

**Honesty on "offline"**: this LAN node had no dependency on the main
dev stack and made no calls to any external service during the test —
but this container has no ability to physically or OS-level disconnect
networking (no root, no `iptables`, shared sandbox). This is **local
isolation, not physical network disconnection**. It proves the LAN
node's own code path works correctly end-to-end; it does not prove what
happens at the OS/network layer if a real machine's internet uplink is
severed mid-exam. That remains untested here — see §4.

### Task 2 — PWA cache cross-tenant data-leak

**What was found**: `next.config.ts`'s Workbox `runtimeCaching` matcher
for API routes checked only `url.pathname.startsWith('/api/')`, with no
origin check. Browser Cache Storage keys entries by request URL alone,
never by the `Authorization` header used to identify the user/tenant. On
a shared device (a school's own lab machine, the actual target deployment
for this product), if the app had ever been pointed at a different origin
or a second tenant's response reused a path, a cached response could be
served to the wrong user.

**Fix**: `src/lib/pwa/apiCacheMatcher.ts`'s `shouldCacheAsApiRoute()` now
requires `url.origin === currentOrigin` in addition to the path check.

**Verification**: new unit tests (`apiCacheMatcher.test.ts`) covering
same-origin/cross-origin/auth-path cases, plus — critically — a real
**production** build (`npm run build`, not dev mode, since `next-pwa` is
disabled in dev) to confirm the generated `public/sw.js` actually calls
the fixed matcher function with the real page origin, verified by
grepping the built, minified service-worker output directly rather than
trusting the source change alone.

### Task 3 — Database backup/restore tooling

New `scripts/backup-school-node.sh` (pg_dump, gzip, write-then-rename,
retention pruning) and `scripts/restore-school-node.sh` (typed
confirmation, stop/restore/restart), plus `GET /admin/backups` to list
available backups.

**Verified**: the actual `pg_dump`/`gunzip`/`psql` pipeline, run directly
against a real Postgres instance — created data, backed it up, dropped
it, restored it, confirmed the data came back correctly.

**Not verified, and disclosed as such in the script's own commit and in
`README-SCHOOL-NODE.md`**: the `docker compose exec` wrapper the scripts
use in a real deployment. **Docker cannot run inside this sandboxed
container at all** (`docker.sock` does not exist here; starting the
Docker daemon fails with an `ulimit`/permission error that is a hard
property of this environment, not something fixable from inside a
session). This is a real, confirmed gap: the underlying dump/restore
logic is proven correct, the Docker Compose plumbing around it is not.

### Task 4 — Native Windows service for the LAN server

New `cmd/windows-service/main_windows.go`: wraps the existing
`cbt-api-server` binary as a real Windows Service (install/remove/run via
`golang.org/x/sys/windows/svc`), with crash-recovery actions and proper
SCM status transitions.

**Verified**: cross-compiles cleanly (`GOOS=windows go build`); the
supervision-loop logic (`Execute()`) itself runs correctly against the
real built `cbt-api-server.exe` under **Wine** (a Windows API
compatibility layer on Linux — not a Windows kernel), confirmed it starts
the child process, serves real HTTP, and is reaped on stop.

**Not verified, and disclosed plainly in `WINDOWS-SERVER.md`**: real
Windows Service Control Manager integration (`install`/`start`/`stop` via
`sc.exe`/`services.msc`), Windows Event Log output, or boot-time
autostart. Wine does not implement the SCM at all — there is no way to
exercise that from this sandbox. **This needs a real Windows machine**
before it can be called production-verified.

### Task 5 — Electron runtime LAN server address configuration

Previously the Electron installer baked one server URL in at build time —
one installer could only ever serve one school. Added a first-run setup
screen (`electron/setup.html`) that lets each school enter and test its
own LAN Node (or cloud) address, persisted locally, changeable later via
a menu item.

**Verified live**: launched the real Electron app (not a mock, not a
unit test) under `xvfb-run` using Playwright's Node-only `_electron`
driver, with narrow verification-only environment overrides
(`CBT_FORCE_SETUP_SCREEN`, `CBT_USER_DATA_DIR` — never read by the real
packaged app's normal path). Confirmed: the setup screen renders with the
correct title and prefilled address (screenshot below); Test Connection
succeeds against the real running LAN node from Task 1 (port 8081,
independently health-checked first) and fails cleanly with a clear
message against an unreachable port; clicking Connect navigates the main
window to the saved address and persists
`{"serverUrl":"http://127.0.0.1:8081"}` to the OS user-data directory.

![Setup screen, successful connection test](electron-setup-screen-success.png)

70/70 Vitest tests pass (13 new); `tsc --noEmit` clean.

**Not verified**: a real packaged Windows installer running this flow
(same Windows-machine gap as Task 4).

## 3. Installation instructions (as currently implemented)

### School LAN Node (Windows server machine)

1. Build both binaries for Windows:
   `GOOS=windows GOARCH=amd64 go build -o cbt-api-server.exe ./cmd/server`
   `GOOS=windows GOARCH=amd64 go build -o cbt-windows-service.exe ./cmd/windows-service`
2. Copy both `.exe` files into the same directory on the server machine,
   alongside a `service.env` file (see `.env.school-node.example` for the
   required keys: `DATABASE_URL`, `JWT_SECRET`, `ENCRYPTION_MASTER_KEY`,
   `REDIS_URL`, etc. — generate real secrets with
   `scripts/generate-school-node-secrets.sh`, never reuse the example
   values).
3. From an elevated prompt in that directory: `cbt-windows-service.exe run`
   to verify it starts correctly in the foreground first, then
   `cbt-windows-service.exe install` and `net start CBTSchoolNode` to run
   it as a real Windows service.
4. Full detail, troubleshooting, and the backup/restore procedure: see
   `WINDOWS-SERVER.md` and `README-SCHOOL-NODE.md` in `cbt-api`.

### Student/teacher desktop client (Electron)

1. Install the built app (see §4 below regarding installer-publishing
   status).
2. On first launch, the setup screen asks for the school's server
   address — the LAN Node's `http://<server-ip>:<port>`, found on the
   server machine or from the school's IT admin.
3. Click Test Connection to confirm reachability, then Connect. The
   address is saved and reused on every future launch; it can be changed
   later from File → Change Server Address.

## 4. Remaining limitations — stated plainly, not glossed over

1. **No true physical network-disconnection test was performed.** This
   sandboxed cloud container has no OS-level network control (no root,
   no `iptables`) and no second physical machine. Every "offline"
   verification in this round used **local isolation** (a LAN node with
   no external dependency called during the test, or a deliberately
   unreachable port to simulate failure) — a reasonable proxy, but not
   the same as unplugging a real network cable. Confirming the full
   workflow with internet physically disconnected, as your instructions
   require, needs real hardware and has not been done.
2. **Docker-based deployment is unverified.** Docker cannot run in this
   container at all. The backup/restore *logic* is proven; the
   `docker-compose.school-node.yml` wrapper around it is not.
3. **Windows Service Control Manager integration is unverified.** Wine
   proves the application logic; it cannot exercise the real SCM, Event
   Log, or boot-time autostart. Needs a real Windows machine.
4. **The packaged Windows Electron installer was not installed and run
   end-to-end on real Windows** — the setup-screen flow was verified via
   Playwright driving the real Electron runtime in this Linux sandbox,
   which exercises the same application code but not the real installer
   package or a real Windows OS.
5. ~~No dedicated health-monitoring dashboard UI was built this round.~~
   **Done in a later round** — see `/dashboard/admin/node-health`
   (`elias-cbt-portal` `df40426`) and `GET /admin/system/health`
   (`cbt-api` `512b839`). Connected-client count and disk/storage usage
   are still not tracked — no session-tracking infrastructure exists to
   back a real client count, and this page says so explicitly rather
   than fabricating one.
6. **The full Section 8 end-to-end checklist from your spec is only
   partially exercised.** Covered this round: question creation, bulk
   publish, student exam access, answer submission, auto-grading
   (including the bug fix), results, backup/restore of real data. Not
   yet done: a from-scratch install with no prior state, multiple
   simultaneous students on one LAN node, server/app restart mid-exam
   recovery, and re-verification of school-isolation specifically in a
   two-LAN-node setup.
7. **Installer publishing to GitHub Releases remains paused**, per your
   explicit instruction to prioritize the offline/LAN workflow first —
   `electron-builder.yml`'s publish config exists (`6c1f141`) but no
   release has been created or uploaded.

## 5. Round 3 — the remaining Section 8 checklist items, actually run

This section covers the four checklist items §4 listed as "not yet done":
a from-scratch install, multiple simultaneous students, restart/recovery,
and cross-node isolation. All four were run live against a genuinely
fresh container in this environment (it came up with no running services
and no data at all — a real, not staged, blank slate) rather than reusing
state from the earlier rounds.

### 5a. Fresh from-scratch install, no prior state

Started from an empty Postgres (two brand-new databases created,
`cbt_lan_node1`/`cbt_lan_node2`) and a `cbt-api-server` binary built fresh
from source (`go build ./cmd/server`). Booted with `AUTO_MIGRATE=true`
against the empty database: the log shows every table being created from
scratch (`CREATE TABLE "tenant_audit_logs"`, etc. — dozens of `CREATE
TABLE`/`CREATE INDEX` statements, none pre-existing). `ADMIN_EMAIL`/
`ADMIN_PASSWORD` auto-created the first admin account on that same boot,
confirmed by logging in with it and getting a real, valid JWT back.
**This is the closest this sandbox can get to a true "install on a blank
machine" test** — the one piece it still can't cover is a real Windows
install (§4.3/§4.4 still apply).

### 5b. Multiple simultaneous students

Built a full academic chain by hand through the real API (session, term,
class level, class arm, class, subject), registered a teacher and three
students, created and published 6 questions, created and published an
exam, assigned it to all three students. Then ran all three students'
**entire exam flow — login, start, answer, submit — concurrently** (three
backgrounded shell processes, not sequential calls) against the one LAN
node:

| Student | Answers | Score | Passed |
|---|---|---|---|
| 1 | all correct | 60/60 | yes |
| 2 | 3 correct, 3 wrong | 30/60 (exact pass-mark boundary) | yes |
| 3 | all wrong | 0/60 | no |

Each got a distinct attempt ID, the correct score for their own answers,
and no cross-contamination between concurrent requests — grading held up
correctly under real concurrent load, including the exact boundary case
(pass mark 30, student 2 scored exactly 30).

### 5c. Server restart mid-exam recovery

Started a new attempt, saved 3 of 6 answers (confirmed via the API:
`answered: 3, unanswered: 3, percentage: 50`), then `kill -9`'d the LAN
node process outright and confirmed it was actually down (`curl`:
connection refused). Restarted the same binary against the same
database. Fetched the attempt's state back: **all 3 pre-crash answers
were intact**, byte-for-byte (same selected options), the timer had
continued correctly, and the attempt was still `in_progress`. Answered
the remaining 3 questions and submitted: final score was **40/60** —
exactly what the 6 answers (3 before the crash, 3 after) should produce,
confirming zero data loss across a genuine process crash and restart.

This is the closest proxy available in this sandbox for "the LAN server
loses power/network mid-exam and comes back" — it exercises the real
recovery path (database-backed attempt state surviving a process crash)
but does not exercise what a student's own client does while the server
is unreachable (that's a separate, already-covered concern — see the
PWA offline-continuity testing referenced in
`OFFLINE-FIRST-COMPLETION-REPORT.md`).

### 5d. Cross-node isolation re-verification

Stood up a second, fully independent LAN node (`cbt_lan_node2`, a
different JWT secret, a different port) alongside the first. Confirmed,
live:

- A real JWT issued by Node 1 for a real logged-in student returned
  **401 "Invalid or expired token"** when presented to Node 2 (different
  `JWT_SECRET`s mean the signature simply doesn't verify) — while the
  same token continued to work normally against Node 1 itself (control).
- Node 1's school registration code was rejected by Node 2
  (`"invalid school code"`), and a direct query against Node 2's own
  database confirmed **zero rows** referencing Node 1's school or users —
  not just an API-level rejection, but genuinely no shared data at any
  level, as expected for two separate database instances.

### 5e. A real bug found and fixed along the way

Running 6b through the real HTTP API (rather than asserting it from
reading code) surfaced a genuine defect: `POST /exams/start` — a
secondary route, not the one the live frontend calls, but still real
public API surface — stored a user's ID into the new exam attempt using
the wrong ID type (the `User` table's ID instead of the separate
`Student` table's own ID). Every subsequent save-answer/submit call on
an attempt started this way then failed its ownership check
("unauthorized"), always, since the two ID spaces never match. Confirmed
live (reproduced the failure, applied the fix, re-ran the exact same
flow, confirmed success), fixed in `30e459c`, `go build`/`go vet`/
`go test ./internal/cbt/...` all pass. No handler-level test harness
exists yet for this package to add an automated regression test against
— noted as a real gap rather than skipped silently.

### 5f. Updated limitations after Round 3

Everything in §4 still applies except item 6, which is now done. Net
new, carried forward:

- No handler-level test coverage exists for `internal/cbt/handler` — the
  bug in §5e was caught by live testing, not a regression test, and
  nothing currently guards against it recurring.
- The restart-recovery test (§5c) proves the *server-side* database
  recovery path; it does not exercise what a real client (browser tab,
  Electron window) does while the server is briefly unreachable, which
  is a different, already-tested concern (see
  `OFFLINE-FIRST-COMPLETION-REPORT.md` §3e/§6 for that side).
- Two Section 8 items remain untested even after this round: a true
  multi-device setup (two or more separate physical machines, one
  running the LAN node and others as separate client devices on the same
  LAN) and re-verification of the PWA/Electron client specifically with
  its own network connection (not the server's) physically cut — both
  need real hardware this sandbox doesn't have.

## 6. Round 4 — bulk-import re-verification (CSV/Excel/Word/TXT)

A later round re-verified bulk question import against the real HTTP
API rather than trusting the existing unit tests alone, per your
instruction to re-check this specifically.

**Method**: for each of the four formats the spec names (CSV, Excel,
Word/DOCX, plus TXT, which the system also supports), downloaded the
system's own official template via the real endpoint
(`GET /questions/templates/{csv,excel,docx,txt}`) and re-uploaded it,
**unmodified**, through the real `POST /questions/bulk-upload` endpoint
against a live LAN node - the strongest test available, since a human
teacher following the documented workflow would do exactly this.

**Result**: every format's own official sample essay question was
rejected, 100% of the time, for two compounding reasons:

1. No import format (CSV, Excel, DOCX, TXT) ever populates a structured
   rubric, but essay questions were hard-required to have one, with no
   fallback.
2. `DOCXParser`/`TXTParser`'s type-detection logic had no path that
   ever produced "essay" - any question with no detected A-D options
   fell through to "single choice" with zero options, which then failed
   for lacking options.

**Fixed** (`cbt-api` `7e73c32`): essay rows now get a real default
rubric synthesized from their own explanation/model-answer text (never
fabricated), and the DOCX/TXT parsers correctly classify an option-less
question as an essay. A third, subtler bug found while building that
fix - the model-answer text was being mangled down to a single stray
letter before it could be used - was fixed in the same commit.

**Verified with the standard red/green discipline**: reverted the
parser fix, confirmed two new regression tests fail for the exact
stated reasons (wrong type, then wrong/empty explanation), restored the
fix, confirmed both pass. Then re-verified live end-to-end on the real
LAN node a second time: all four official templates now import their
essay questions correctly, confirmed by querying the resulting database
rows directly - the full model-answer text appears as both the
question's explanation and a real rubric criterion, not a stray letter
or an empty field.

**Also confirmed working, unaffected by this bug**: single-choice,
multiple-choice, true/false, and fill-in-the-blank questions bulk-import
correctly in every format tested (they always did - only essay
questions were broken).

## 7. Round 5 — JSON, RTF, PDF, and legacy XLS bulk-import re-verification

Closes the gap §6 left open. No official downloadable template exists
for these four formats (`GET /questions/templates/{format}` only
supports csv/excel/docx/txt), so the method differed slightly per
format, but every file used was genuine, not hand-faked bytes:

- **XLS**: the real Excel template downloaded via the live API,
  converted to true legacy Excel 97 binary format with
  `soffice --headless --convert-to xls` (confirmed via `file`: a real
  "Composite Document File V2... MS Excel 97" binary).
- **RTF**: the real DOCX template converted the same way
  (`--convert-to rtf`), confirmed a real "Rich Text Format data" file.
- **PDF**: the real DOCX template converted to a genuine text-based PDF
  (`--convert-to pdf`), confirmed a real "PDF document, version 1.7".
- **JSON**: hand-written against `JSONQuestion`'s real field names in
  `question_dto.go` (no generator exists for this format), with one MCQ
  and one essay question.

All four uploaded through the real `POST /questions/bulk-upload`
against a live LAN node, no code changes needed or made this round:

| Format | Result |
|---|---|
| XLS | 2/2 rows succeeded (MCQ + essay), confirmed via DB query |
| RTF | Both real sample questions succeeded; 2 of 4 total rows failed, but those were fragments of the template's own instructions paragraph (RTF's paragraph-break conversion split it differently than DOCX's), honestly rejected with a clear message - not silent data loss, and not a new bug |
| PDF | 2/2 real questions succeeded (MCQ + essay); the usual single instructions-paragraph pseudo-row was honestly rejected, matching DOCX/TXT's existing documented behavior |
| JSON | 2/2 succeeded (MCQ + essay) cleanly, no failures at all |

Every essay question in every format landed with its full, correct
model-answer text as both `explanation` and a real synthesized rubric
criterion - confirmed by querying the resulting rows directly, not
just reading the API response.

**Why no code change was needed**: reading the parsers first (rather
than guessing) showed RTF and PDF both parse by extracting plain text
and handing it to the exact same `TXTParser.parseQuestionsFromText`
already fixed in §6, and XLS parses by handing its rows to the exact
same `ExcelParser.parseRow` Excel already uses - so all three inherited
the fix automatically. JSON was never susceptible to the type-detection
bug in the first place (its question type is an explicit input field,
never auto-detected), and already benefits from the service-layer
rubric-synthesis fix since its schema always had an `explanation`
field. This was confirmed by live-testing each format rather than
trusting that code-reading conclusion alone.

**Still not covered**: none of the nine supported bulk-import formats
remain unverified against the real HTTP endpoint as of this round.

## 8. Round 6 — report-printing re-verification

Started by actually loading a real completed exam's result page rather
than trusting it worked, per your instruction. It crashed immediately:
*"Unable to Load Results / Cannot read properties of undefined
(reading 'correctAnswers')"*.

### 8a. A real crash blocking every exam result page

Root cause: the real backend response (`GET
/student/exams/result/:examId`) is snake_case throughout
(`result_summary`, `exam_title`, `total_score`, ...), confirmed
directly against cbt-api's own struct tags and a real response. But
`elias-cbt-portal`'s `src/types/student.ts` declared this response as
**camelCase** - a type that was never checked against a real response
- and two pages (`result/page.tsx`,
`result/[resultId]/page.tsx`) trusted it, reading
`raw.resultSummary.correctAnswers`. That key doesn't exist (the real
one is `raw.result_summary`), so every load of either page crashed
before rendering anything. A third page using the same endpoint
(`review/page.tsx`) was never affected - it already read the real
snake_case fields with a camelCase fallback, the pattern applied to
fix the other two.

Fixed in `elias-cbt-portal` `ed18af8`. Verified live: logged in as a
real student with a real completed exam, both pages now render the
real score (40/60, 66.7%, grade B) correctly.

### 8b. Printing produced the full dashboard chrome, not a report

Separately, **zero `@media print` rules existed anywhere in this
codebase** (confirmed by grepping the whole `src/` tree) - none of the
six `window.print()` call sites in the app would have produced a
usable printout. Confirmed live by emulating print media on the (now-
fixed) result page: the printed render was pixel-identical to the
screen render - full dark sidebar, topbar, confetti canvas, and action
buttons included.

Fixed in `elias-cbt-portal` `3788592`: a real print stylesheet that
hides the actual Sidebar/Header components specifically (via a
`data-app-chrome` marker, not a bare `aside`/`header` tag selector that
could also hide a report's own masthead), hides page-specific controls
via a new `.no-print` class, and resets the content area to a plain,
full-width, ink-friendly layout. Applied across both result pages, the
review page, the parent termly-results page, and the admin exam-paper
preview.

The exam-paper preview (`PreviewExamModal`, the "examination papers...
for printing" feature) needed an additional fix beyond the global
rules: it's a fixed, backdrop-blurred, height-capped overlay with an
internal scroll area, so printing it unmodified would print a dark
backdrop and cut the exam paper off at the modal's own on-screen
scroll height instead of printing every question. Added specific
overrides to reset position/background/height/overflow for print, plus
a print-only title heading, and marked the dashboard list page behind
the modal as non-printable too (it was printing stacked alongside the
actual exam paper).

**Verified live for both printable-report types the spec names** -
screenshots captured before and after each fix:
- A real exam result: confetti/sidebar/topbar/buttons gone, clean
  full-width report remains.
- A real exam paper (6 real questions): backdrop and underlying
  dashboard page gone, all 6 questions print instead of being cut off.

`npx tsc --noEmit` and `npx vitest run` (70/70) pass after both fixes.

### 8c. Known remaining gaps in report printing

1. **The student `report-card` page
   (`/dashboard/student/results/report-card`) is still entirely mock
   data** - a pre-existing, already-disclosed condition (the page's own
   comment says "In production, fetch from API" with the real call
   commented out). The real backend endpoint
   (`GET /student/exams/report-card`) was confirmed live this round to
   work correctly and return real, correct, snake_case termly data for
   a real student - but the frontend page was not wired up to it. This
   is a larger feature-completion task (the page also has no real
   term/session selection UI yet, just a hardcoded placeholder) rather
   than a bug fix, and was not attempted this round to avoid a rushed,
   inadequately-verified integration under time pressure. Recommended
   as focused follow-up work.
2. **`src/types/student.ts`'s `StudentExamResultResponse` type is still
   wrong** (declared camelCase, real shape is snake_case). Both broken
   pages were fixed at the call site with defensive fallbacks rather
   than correcting the shared type, since the type is also referenced
   by mock adapters and other services not evaluated this round - a
   wider, riskier change than this round's scope. The type lie remains
   a latent trap for any future page that trusts it naively, same as
   the two just fixed.
3. **A pre-existing, unrelated display quirk noticed in passing**: a
   wrong-answer row's "Correct:" label shows blank on the
   `result/[resultId]` page - the backend's `GetStudentExamResult`
   populates `CorrectAnswer` from the legacy single-string field only,
   which is empty for modern option-key-based questions (the same
   `CorrectOptionKeys` vs `CorrectAnswer` distinction behind the
   auto-grading bug fixed earlier in this engagement, `5afdb10`). Not
   fixed this round - noted for follow-up.

## 9. What to do next, in priority order

1. Get access to a real Windows machine to close the two Windows-specific
   gaps (§4.3, §4.4) and the real multi-device/client-network-cut gaps
   (§5f) — nothing further can be verified for those from this sandbox.
2. Wire the student report-card page to the real, already-working
   backend endpoint (§8c.1).
3. Add handler-level test coverage for `internal/cbt/handler` so a bug
   like §5e's can't silently recur.
4. Resume the Windows-installer GitHub Releases publish.
