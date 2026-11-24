# Gabugu Docker Deployment

Gabugu production images are built from the reviewed checkout. The build compiles the frontend, embeds it into the Go server, and never pulls an upstream application image.

## Required Secrets

Create `platform/deploy/.env` from `.env.example` and set independent values for:

- `POSTGRES_PASSWORD`
- `JWT_SECRET`
- `GABUGU_API_KEY_HMAC_KEY`
- `GABUGU_UPSTREAM_ENCRYPTION_KEY`

Generate each Gabugu key independently with `openssl rand -hex 32`. Do not reuse the JWT or TOTP key.

## Local Reviewed Build

```bash
cd platform/deploy
cp .env.example .env
chmod 600 .env
docker compose --env-file .env build sub2api
docker compose --env-file .env up -d
```

The application is available only on `127.0.0.1:18081` by default. Set `GABUGU_PORT=18082` for the alternate slot. Container port `8080` is never published directly on the host.

## Blue-Green Deployment

The production blue-green stack, ordered deployment script, and Nginx template live in `platform/backend`. Replace the example host and TLS paths in `scripts/nginx-gabugu.conf.example`, install it under Nginx's `http` context, and create `/etc/nginx/gabugu` before the first deployment. The script writes the active upstream include only after health succeeds.

```bash
GABUGU_ENV_FILE="$PWD/platform/deploy/.env" \
  platform/backend/scripts/deploy.sh deploy

platform/backend/scripts/deploy.sh rollback
```

The deploy command creates a PostgreSQL backup before migrations, builds the reviewed image, migrates and starts the inactive slot, waits for `/health`, then reloads Nginx at the explicit traffic-switch boundary. Rollback switches traffic only to the previously healthy slot; database restore remains an operator-controlled recovery action.
