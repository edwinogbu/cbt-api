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
report covers two rounds of work: the five initial tasks below (§1-§4),
and a follow-up round (§5) that ran the remaining items from your
Section 8 end-to-end checklist.

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
5. **No dedicated health-monitoring dashboard UI was built this round.**
   The backend `GET /admin/backups` endpoint exists; there is no page
   showing LAN-node uptime, connected-client count, or live health status.
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

## 6. What to do next, in priority order

1. Get access to a real Windows machine to close the two Windows-specific
   gaps (§4.3, §4.4) and the real multi-device/client-network-cut gaps
   (§5f) — nothing further can be verified for those from this sandbox.
2. Build the health-monitoring dashboard page against the existing
   backend endpoints.
3. Add handler-level test coverage for `internal/cbt/handler` so a bug
   like §5e's can't silently recur.
4. Resume the Windows-installer GitHub Releases publish, once the above
   is closed out.
