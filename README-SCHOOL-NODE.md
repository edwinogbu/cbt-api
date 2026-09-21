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

This prints a fresh `DB_PASSWORD`, `JWT_SECRET`, and `JWT_REFRESH_SECRET`.
Copy the template and paste them in:

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
`DB_PASSWORD`, `JWT_SECRET`, or `JWT_REFRESH_SECRET` aren't set - there is no
insecure fallback baked into the deployment config, and the binary itself
now refuses to boot in `ENVIRONMENT=production` with the placeholder JWT
secret checked into the repo (see `config/config.go`), so a node can't come
up silently misconfigured either way.

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
  node needs to be reachable outside the school building.
- **Backups**: `cbt_node_pgdata` is a named Docker volume. Set up your own
  periodic `pg_dump` (or volume snapshot) - nothing here backs it up for you.
- **Multi-node**: this guide is for one node. Repeat steps 1-3 per school,
  each with its own generated secrets and its own Postgres/Redis - nodes do
  not share a database.

## Troubleshooting

- **Container `cbt-api` restarts in a loop, logs show
  `FATAL: JWT_SECRET must be set...`**: step 1 wasn't completed, or
  `.env.school-node` still has the placeholder `CHANGE_ME...` value. Re-run
  the generator script and check the file was actually passed via
  `--env-file`.
- **`docker compose ... config` errors with
  `required variable DB_PASSWORD is missing a value`**: same as above - a
  required secret isn't set. This is the compose file's env-var guard
  working as intended, not a bug.
- **`cbt-api` can't reach Postgres**: check `docker compose -f
  docker-compose.school-node.yml logs postgres` - the `cbt-api` container
  waits for Postgres's healthcheck before starting, so a Postgres startup
  failure will show there first.
