# Running a School CBT Node natively on Windows

This is an alternative to the Docker-based deployment in
`README-SCHOOL-NODE.md` for a school whose server machine is an
ordinary Windows PC rather than something that can run Docker/WSL2
comfortably (older hardware, no virtualization enabled in BIOS, IT
staff more comfortable with native Windows services than containers).
It runs the **exact same `cbt-api` application** - nothing about exam
logic, question banks, authentication, or data isolation differs
between this and the Docker path. The only difference is how the
process is started, supervised, and restarted on boot: a native Windows
Service (via `cmd/windows-service`, built on
`golang.org/x/sys/windows/svc`) instead of `docker compose`.

## What you need

- A Windows 10/11 (or Windows Server 2016+) machine on the school's own
  network. No internet connection is required to run exams once this
  is installed - only to install it the first time (downloading
  PostgreSQL and Redis) unless you obtain those offline installers
  separately.
- **PostgreSQL 14+** and **Redis** (or a Redis-compatible server, e.g.
  Memurai on Windows) installed as their own Windows services. This
  guide doesn't bundle them - they're mature, widely-packaged Windows
  installers in their own right (postgresql.org/download/windows,
  Memurai or redis-windows community builds), and reusing those rather
  than re-packaging a database engine ourselves is the safer path.
- Administrator access, to install the Windows service.

### Minimum hardware (documented, not independently load-tested on real
Windows hardware in this pass - see "What was and wasn't verified" below)

| | Minimum | Recommended (200+ concurrent students) |
|---|---|---|
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Disk | 20 GB free (SSD strongly preferred) | 50+ GB SSD |
| Network | 100 Mbps LAN/Wi-Fi | Gigabit wired for the server's own uplink to the school switch |

These are the same practical minimums any small Postgres+Go-web-service
deployment needs, not a number specific to exam load testing performed
in this sandbox - this sandbox has no real Windows hardware to run that
load test on (see below).

## Install

1. **Build the two binaries** (done once per release, from a machine
   with Go installed - can be this same Windows machine, or any
   machine cross-compiling for it):

   ```bash
   GOOS=windows GOARCH=amd64 go build -o cbt-api-server.exe ./cmd/server/
   GOOS=windows GOARCH=amd64 go build -o cbt-windows-service.exe ./cmd/windows-service/
   ```

2. **Copy both `.exe` files into the same directory** on the Windows
   machine, e.g. `C:\CBTSchoolNode\`.

3. **Create `C:\CBTSchoolNode\service.env`** - the same environment
   variables `.env.school-node.example` documents for the Docker path,
   one `KEY=VALUE` per line (`#` for comments). At minimum:

   ```
   DATABASE_URL=postgres://cbt_node:YOUR_PASSWORD@localhost:5432/cbt_db?sslmode=disable
   REDIS_ADDR=localhost:6379
   JWT_SECRET=<run scripts/generate-school-node-secrets.sh for a real one>
   JWT_REFRESH_SECRET=<run scripts/generate-school-node-secrets.sh for a real one>
   ENCRYPTION_MASTER_KEY=<run scripts/generate-school-node-secrets.sh for a real one>
   ENVIRONMENT=production
   PORT=8080
   AUTO_MIGRATE=true
   ADMIN_EMAIL=admin@yourschool.example
   ADMIN_PASSWORD=<a real password - creates the first admin account on first boot>
   ```

   `scripts/generate-school-node-secrets.sh` is a bash script - run it
   on any Linux/macOS machine (or WSL, or Git Bash on the same Windows
   box) once to generate the three secret values, then paste them in
   here. The secrets themselves have nothing Linux-specific about them.

4. **Verify it runs** before installing as a service (open PowerShell
   as Administrator in `C:\CBTSchoolNode\`):

   ```powershell
   .\cbt-windows-service.exe run
   ```

   This starts `cbt-api-server.exe` in the foreground with logs
   printed to the console - confirm it connects to Postgres/Redis and
   `http://localhost:8080/api/v1/health` responds, then `Ctrl+C` to
   stop it.

5. **Install as a Windows service**:

   ```powershell
   .\cbt-windows-service.exe install
   net start CBTSchoolNode
   ```

   The service is configured to start automatically on boot and to
   restart itself after a crash (after 5s, then 15s, then 60s, resetting
   that backoff after a day of stable uptime) - set at install time via
   `mgr.SetRecoveryActions`, not something that needs separate
   configuration in the Services console afterward.

6. **Point the frontend at this machine's LAN address**, same as the
   Docker path - see README-SCHOOL-NODE.md section 4.

## Day-to-day operation

- **Status / start / stop**: Windows Services console
  (`services.msc`), or `net start|stop CBTSchoolNode`, or
  `sc query CBTSchoolNode`.
- **Logs**: `C:\CBTSchoolNode\cbt-api-service.log` (the wrapped
  server's stdout/stderr) and the Windows Event Log (Application log,
  source `CBTSchoolNode`) for service-lifecycle events
  (start/stop/crash) specifically.
- **Recovery after an unexpected shutdown**: the service's own
  `StartAutomatic` type means Windows starts it again on the next boot
  with no manual step. `AUTO_MIGRATE=true` means any pending database
  migration also reapplies safely on that restart (the same idempotent
  migration path the Docker deployment uses - nothing Windows-specific
  about it).
- **Uninstall**: `net stop CBTSchoolNode` then
  `.\cbt-windows-service.exe remove`.

## Backups

Same scripts as the Docker path do not apply directly here (they shell
out to `docker compose exec`) - for a native Windows Postgres install,
use `pg_dump`/`pg_restore` directly (PostgreSQL's own Windows installer
includes both on `PATH`), e.g. as a scheduled task via Windows Task
Scheduler running:

```powershell
pg_dump -U cbt_node -d cbt_db --clean --if-exists -f C:\CBTSchoolNode\backups\cbt-node-%date:~-4,4%%date:~-10,2%%date:~-7,2%.sql
```

The underlying dump/restore logic is identical to
`scripts/backup-school-node.sh`'s - see that script and
README-SCHOOL-NODE.md's "Backups and restore" section for the full
reasoning (`--clean --if-exists`, restrictive file permissions, etc.);
only the Docker-exec wrapper differs.

## What was and wasn't verified in this pass

**Verified for real, in this development sandbox:**

- Both `cbt-api-server.exe` and `cbt-windows-service.exe` cross-compile
  cleanly from this Linux sandbox with `GOOS=windows GOARCH=amd64` -
  zero cgo dependencies, zero code changes needed in the application
  itself (`go vet`/`go build ./...` both clean on both platforms from
  the same source tree).
- `cbt-windows-service.exe run` (the foreground/debug mode, not the
  actual Windows-service-managed path) was run for real under Wine
  (a Windows API compatibility layer on Linux - not a substitute for
  real Windows, but real evidence beyond "it compiled"): it correctly
  resolved `cbt-api-server.exe`'s path relative to its own binary,
  loaded `service.env`, spawned the real Windows PE binary as a child
  process, and that child connected to a real Postgres database, ran
  auto-migration, and served `GET /api/v1/health` with a genuine HTTP
  response - confirmed via `curl` against the Windows binary's own
  listening port and via the real OS process tree showing both
  `cbt-windows-service.exe` and the child `cbt-api-server.exe` running.

**NOT verified - genuinely needs real Windows hardware:**

- The actual Windows Service Control Manager integration
  (`install`/`remove`, `net start`/`net stop`, automatic start on boot,
  the configured crash-recovery actions actually triggering). Wine does
  not implement a real SCM, so none of `golang.org/x/sys/windows/svc`'s
  SCM-facing code path (as opposed to the `Execute` supervision loop
  itself, which *was* exercised via the `run` command above) has been
  exercised against an actual Windows kernel.
- Windows Event Log registration/writing (`eventlog.InstallAsEventCreate`,
  `elog.Info`).
- Load behavior under many concurrent students/exams on real hardware -
  the hardware table above is a documented estimate, not a measured
  result from this sandbox.
- A real offline install (no internet reachable at all during setup) -
  this sandbox's own network is proxied/sandboxed, not a genuine
  disconnected Windows LAN.

Run the install steps above against a real Windows machine, with its
network cable/Wi-Fi disabled after the one-time PostgreSQL/Redis
installer downloads, before relying on this operationally for a real
school.
