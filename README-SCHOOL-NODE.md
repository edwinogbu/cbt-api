# Deploying a School CBT Node

A School CBT Node is the same `cbt-api` binary that runs in the cloud, running
instead on a school's own LAN with its own local Postgres and Redis, so the
school can run exams with no dependency on internet reachability. Nothing in
the application code changes between the two - only configuration.

This guide stands up one node with Docker. It assumes Docker and Docker
Compose are installed on the machine you're deploying to (a small server or
even a capable desktop on the school's network works).

## 1. Generate this node's secrets

**Every node needs its own, unique secrets.** Never copy a `.env.school-node`
file from one school's node to another's.

```bash
scripts/generate-school-node-secrets.sh
```

This prints a fresh `DB_PASSWORD`, `JWT_SECRET`, `JWT_REFRESH_SECRET`, and
`ENCRYPTION_MASTER_KEY`. Copy the template and paste them in:

```bash
cp .env.school-node.example .env.school-node
# edit .env.school-node: paste in the generated values, plus this school's
# admin email/password and the LAN port you want the API on.
```

`.env.school-node` now holds real secrets - it's covered by `.gitignore`'s
`.env.*` pattern (with `.env.*.example` explicitly kept trackable so this
template itself stays in the repo); double check `git status` before
committing anything in this directory regardless.

## 2. Build and start the node

```bash
docker compose -f docker-compose.school-node.yml --env-file .env.school-node up -d --build
```

This builds the `cbt-api` image from the `Dockerfile` in this repo (multi-stage,
`CGO_ENABLED=0`, runs as a non-root user) and starts three containers on an
isolated network: `postgres`, `redis`, and `cbt-api`. The compose file
deliberately refuses to start (`required variable ... is missing`) if
`DB_PASSWORD`, `JWT_SECRET`, `JWT_REFRESH_SECRET`, or `ENCRYPTION_MASTER_KEY`
aren't set - there is no insecure fallback baked into the deployment config,
and the binary itself now refuses to boot in `ENVIRONMENT=production` with
the placeholder JWT secret or encryption key checked into the repo (see
`config/config.go` and `pkg/utils/encryption.go`), so a node can't come up
silently misconfigured either way.

On first boot, `AUTO_MIGRATE=true` runs the schema migration against the
fresh local Postgres, and if `ADMIN_EMAIL`/`ADMIN_PASSWORD` are set, the
first admin account for this school is created automatically.

## 3. Confirm it's up

```bash
curl http://localhost:8090/api/v1/health   # or whatever SCHOOL_NODE_PORT you set
```

Point any device on the school's LAN at `http://<this-machine's-LAN-IP>:<SCHOOL_NODE_PORT>`
and log in with the admin account from step 1.

## 4. Point the frontend at this node

The `elias-cbt-portal` frontend resolves between a LAN address and a cloud
address automatically (`src/lib/network/ConnectionManager.ts`, added as part
of the offline-first work - see that repo's `NEXT_PUBLIC_LAN_API_URL`). Set
it to this node's LAN address so students, teachers, and admins on this
school's network reach this node instead of the cloud, with the cloud
address kept as the fallback if the node ever goes down.

## What this does NOT cover yet

- **School ↔ Cloud sync**: this node runs standalone. Pushing its
  sessions/submissions/results up to the central cloud instance, and pulling
  down schools/teachers/students/exams/schedules, is separate work (tracked
  as its own phase).
- **TLS**: this compose file serves plain HTTP on the LAN, matching a
  school's internal network. Put a reverse proxy in front of it (with a
  real or self-signed certificate) if the network isn't trusted, or if the
  node needs to be reachable outside the school building. **This is no
  longer optional for a production build of the frontend**: as of the
  elias-cbt-portal LAN transport security fix, `ConnectionManager.ts`
  refuses to use `NEXT_PUBLIC_LAN_API_URL` in a production build unless it
  is `https://` - exam answers must not cross the LAN unencrypted. A
  school-node deployment that still serves plain HTTP on the LAN (as this
  guide's default does) will have LAN sync silently disabled by a
  production frontend build, falling back to Cloud, until a TLS-terminating
  reverse proxy is put in front of this node and `NEXT_PUBLIC_LAN_API_URL`
  is pointed at its `https://` address instead.

  **A public CA will not issue a certificate for this node's address as-is.**
  This node is reached by a private LAN IP (e.g. `192.168.x.x`) or a bare
  local hostname - public CAs (Let's Encrypt included) cannot issue for a
  private IP at all, and can't validate a hostname they can't publicly
  resolve/reach either. Don't assume dropping a reverse proxy in front of
  this node with no further certificate planning gets you a browser-trusted
  `https://` address; pick one of these, matched to what the school
  actually has:

  - **No real domain/DNS available (smallest school, no IT staff)**:
    self-signed certificate on the reverse proxy, root CA distributed to
    every student/staff device once (e.g. via whatever device-management
    tooling the school already uses, or manually on a one-time setup pass).
    No ongoing renewal-automation infrastructure needed, but the trust
    distribution step has to be redone for every new device and doesn't
    scale past a small, IT-light deployment.
  - **School already owns a domain + can manage DNS (recommended default)**:
    split-horizon DNS - point a subdomain (e.g. `node1.yourschool.example`)
    at this node's private LAN IP in the school's *internal* DNS only (the
    public DNS record for that name doesn't need to exist, or can point
    anywhere; LAN clients resolve it internally), and get a real
    Let's Encrypt certificate for that subdomain via the **DNS-01**
    challenge (not HTTP-01 - the node isn't publicly reachable, so HTTP-01
    will fail). DNS-01 needs API access to whoever hosts the domain's DNS
    (Cloudflare, Route53, etc.) to create the challenge TXT record, and a
    renewal job (certbot or similar, run from wherever that DNS-01
    automation lives, not necessarily the node itself) since Let's Encrypt
    certs expire every 90 days. Every device trusts it with zero manual
    steps, since it's a real publicly-trusted cert - this is the
    recommended path once a school has a domain at all.
  - **Multiple nodes across a school group with existing IT/MDM**: a
    private internal CA, root distributed once via the same device
    management the IT team already uses, node certs signed by that CA and
    renewed by the IT team's own tooling. Only worth it at a scale where
    self-signed-per-node trust distribution becomes unmanageable.

  Whichever path is chosen, note the expiry and renewal cadence
  explicitly in that school's own deployment runbook - an expired
  certificate silently reproduces the exact "LAN sync falls back to
  Cloud" behavior the plain-HTTP case does, just later and without the
  original misconfiguration being the obvious first guess.
- **Backups**: `cbt_node_pgdata` is a named Docker volume. Set up your own
  periodic `pg_dump` (or volume snapshot) - nothing here backs it up for you.
- **Multi-node**: this guide is for one node. Repeat steps 1-3 per school,
  each with its own generated secrets and its own Postgres/Redis - nodes do
  not share a database.

## Troubleshooting

- **Container `cbt-api` restarts in a loop, logs show
  `FATAL: JWT_SECRET must be set...`** (or `ENCRYPTION_MASTER_KEY must be
  set...` / `...must be exactly 32 bytes...`): step 1 wasn't completed, or
  `.env.school-node` still has the placeholder `CHANGE_ME...` value, or
  `ENCRYPTION_MASTER_KEY` was edited to something other than exactly 32
  characters. Re-run the generator script and check the file was actually
  passed via `--env-file`.
- **`docker compose ... config` errors with
  `required variable DB_PASSWORD is missing a value`**: same as above - a
  required secret isn't set. This is the compose file's env-var guard
  working as intended, not a bug.
- **`cbt-api` can't reach Postgres**: check `docker compose -f
  docker-compose.school-node.yml logs postgres` - the `cbt-api` container
  waits for Postgres's healthcheck before starting, so a Postgres startup
  failure will show there first.
