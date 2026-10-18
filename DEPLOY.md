# Deploying cbt-api to Render

This backend is stateless - Postgres and Redis are both managed
elsewhere (Neon, Upstash) - so the Render service just runs the API
binary from `Dockerfile`, configured entirely by env vars set in
Render's dashboard.

**Status: live** at https://cbt-api-a2it.onrender.com, deployed from
`render.yaml` at the root of this repo via Render's "Blueprint" flow.
Tables are confirmed created on Neon (including the seeded admin
user), so the one-time migration step below is already done for the
current deployment - it's documented here for future reference
(redeploys to a new Render service, disaster recovery, etc.).

None of this can be verified from the Claude session that prepared
it: this sandbox's network policy blocks `render.com`,
`*.onrender.com`, `neon.tech` (including its HTTP query endpoint, not
just the Postgres wire protocol), and the official Neon CLI outright
- confirmed via several independent failed attempts, not a one-off
glitch. Every command below, and all verification, has to happen from
your own machine or the Render/Neon web dashboards.

## One-time setup (Blueprint deploy)

1. Render dashboard → **New +** → **Blueprint**
2. Connect the `cbt-api` GitHub repo, branch `claude/awesome-einstein-df5zdb`
   (update this once the work merges to `main`)
3. Render reads `render.yaml` and shows a preview: one web service,
   `cbt-api`, Docker runtime, plus four blank secret fields (see
   below) - fill those in, then click **Deploy Blueprint**

`render.yaml` deliberately leaves `DATABASE_URL`, `REDIS_URL`,
`JWT_SECRET`, and `JWT_REFRESH_SECRET` as `sync: false` - Render makes
you type them in by hand rather than storing them in the repo.

## Secrets

| Secret | Where it comes from |
|---|---|
| `DATABASE_URL` | Neon dashboard → your project → Connection string (use the **pooled** one, hostname ending `-pooler...neon.tech`) |
| `REDIS_URL` | Upstash dashboard → your database → connection string (starts `rediss://`) |
| `JWT_SECRET` | Any long random string, e.g. `openssl rand -hex 32` |
| `JWT_REFRESH_SECRET` | A **different** long random string, same way |

Optional (only if you want AI question-generation features working -
the server falls back gracefully without them): `OPENAI_API_KEY`,
`ANTHROPIC_API_KEY`, `GEMINI_API_KEY`. Add these in the service's
**Environment** tab same as the required ones.

## First deploy (runs the schema migration once)

The Neon database starts empty - `AUTO_MIGRATE=true` makes the server
create all tables on boot via GORM's AutoMigrate.

1. In the service's **Environment** tab, add/edit `AUTO_MIGRATE` to `true`, save
   (triggers an automatic redeploy)
2. Watch the **Logs** tab for `✅ Database connected` then
   `✅ Synced N tables`
3. Once confirmed (check Neon's **Tables** page too, for the actual
   table list), set `AUTO_MIGRATE` back to `false` and save again

## Every deploy after that

With `autoDeploy: false` in `render.yaml`, pushing to GitHub does
**not** redeploy on its own - that's intentional, so a CI build/test
gate runs first. Two ways to trigger a deploy:

- **Manual**: Render dashboard → the service → **Manual Deploy** → "Deploy latest commit"
- **Automatic via GitHub Actions**: see `.github/workflows/deploy-render.yml` -
  runs on every push, builds and `go vet`s the code, and only on
  success calls Render's Deploy Hook. Requires a GitHub repo secret
  `RENDER_DEPLOY_HOOK_URL` (service → **Settings** tab → "Deploy Hook" → copy
  the URL → GitHub repo → Settings → Secrets and variables → Actions →
  New repository secret).

## Verify it's live

```bash
curl https://cbt-api-a2it.onrender.com/api/v1/health
```
or just open that URL in a browser.

## Cold starts

Render's free tier spins the service down after a period of
inactivity; the next request after that pays a cold-start delay
(up to ~50s). This is a known, accepted tradeoff of the free tier,
not a bug - upgrading the Render plan removes it if it becomes a
problem.

## Wiring the frontend to it

Already done in `elias-cbt-portal/.env.local` and should also be set
in the frontend's Vercel project settings (Environment Variables):

```
NEXT_PUBLIC_CLOUD_API_URL=https://cbt-api-a2it.onrender.com
NEXT_PUBLIC_API_URL=https://cbt-api-a2it.onrender.com/api/v1
```

CORS is currently wide open (`AllowOrigins: []string{"*"}` in
`cmd/server/main.go`) so no backend-side CORS config is needed for
Vercel to call it - that's a pre-existing choice, not something this
deploy changes, and worth tightening later as a separate security
task.
