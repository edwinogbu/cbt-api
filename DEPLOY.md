# Deploying cbt-api to Fly.io

This backend is stateless - Postgres and Redis are both managed
elsewhere (Neon, Upstash) - so the Fly app itself just runs the API
binary from `Dockerfile`, configured entirely by `fly secrets`.

None of this can be run from the Claude session that prepared it: this
sandbox's network policy blocks `fly.io`, `neon.tech`, and
`upstash.io` outright, so every command below has to be run from your
own machine (or the Fly.io web dashboard, where it offers an
equivalent).

## Prerequisites

- flyctl installed: https://fly.io/docs/flyctl/install/
- `fly auth login` (or `fly auth token` if you're scripting this from
  somewhere without a browser)

## One-time setup

```bash
cd cbt-api
fly launch --no-deploy --copy-config
```

This registers the app against your Fly account using the `fly.toml`
already in this repo. Pick "no" if it asks to create a Postgres or
Redis instance - we're using Neon and Upstash instead. If the app name
`cbt-api` in `fly.toml` is already taken globally, `fly launch` will
prompt you for a different one - either is fine, just note it since
it becomes the app's URL (`https://<name>.fly.dev`).

## Secrets

Set every one of these before the first deploy (`fly secrets set` can
take multiple `KEY=value` pairs in one call):

| Secret | Where it comes from |
|---|---|
| `DATABASE_URL` | Neon dashboard → your project → Connection string (use the **pooled** one) |
| `REDIS_URL` | Upstash dashboard → your database → connection string (starts `rediss://`) |
| `JWT_SECRET` | Generate: `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET` | Generate a **different** value the same way |

```bash
fly secrets set \
  DATABASE_URL="postgresql://neondb_owner:...@...-pooler...neon.tech/neondb?sslmode=require&channel_binding=require" \
  REDIS_URL="rediss://default:...@....upstash.io:6379" \
  JWT_SECRET="$(openssl rand -hex 32)" \
  JWT_REFRESH_SECRET="$(openssl rand -hex 32)"
```

Optional (only if you want AI question-generation features working -
the server falls back gracefully without them):
`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`.

## First deploy (runs the schema migration once)

The Neon database starts empty - `AUTO_MIGRATE=true` makes the server
create all tables on boot via GORM's AutoMigrate.

```bash
fly secrets set AUTO_MIGRATE=true
fly deploy
```

Watch the deploy logs (`fly logs`) for `✅ Synced N tables`. Once
you've confirmed that, turn migration back off so it doesn't re-run
(harmless if it does, but unnecessary) on every future deploy:

```bash
fly secrets set AUTO_MIGRATE=false
```

## Every deploy after that

```bash
fly deploy
```

## Verify it's live

```bash
curl https://<your-app-name>.fly.dev/api/v1/health
```

## Wiring the frontend to it

Once deployed, set this in the frontend's Vercel project settings
(Environment Variables) and in `elias-cbt-portal/.env.local` for local
dev against the live backend:

```
NEXT_PUBLIC_CLOUD_API_URL=https://<your-app-name>.fly.dev
NEXT_PUBLIC_API_URL=https://<your-app-name>.fly.dev/api/v1
```

CORS is currently wide open (`AllowOrigins: []string{"*"}` in
`cmd/server/main.go`) so no backend-side CORS config is needed for
Vercel to call it - that's a pre-existing choice, not something this
deploy changes, and worth tightening later as a separate security
task.
