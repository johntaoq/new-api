# GHCR + Azure40 Deployment

This repository already has a GitHub fork at `git@github.com:johntaoq/new-api.git`.
The recommended production path is:

1. Push code to GitHub.
2. Let GitHub Actions build and publish `ghcr.io/johntaoq/new-api`.
3. Let Azure40 pull the published image and restart the container.

## Workflows

- `.github/workflows/ci.yml`
  Runs frontend build, backend build, and a Docker image build on pull requests and pushes to `main` and `codex/**`.
- `.github/workflows/publish-ghcr.yml`
  Publishes the production image to GHCR on `main`, tags, and manual dispatch.
- `.github/workflows/deploy-azure40.yml`
  Manually deploys a chosen image tag to Azure40 over SSH.

## GitHub repo settings

Enable these repository settings:

1. `Settings -> Actions -> General -> Workflow permissions`
   Set to `Read and write permissions`.
2. `Settings -> Secrets and variables -> Actions -> Variables`
   Add `VITE_STUDIO_LAUNCH_URL=https://www.unikeyx.com/_studio/launch`
3. `Settings -> Secrets and variables -> Actions -> Secrets`
   Add:
   - `AZURE40_HOST`
   - `AZURE40_USER`
   - `AZURE40_SSH_KEY`
   - `GHCR_USERNAME` (optional if GHCR package is public)
   - `GHCR_TOKEN` (optional if GHCR package is public)

## Azure40 layout

Recommended server path:

```bash
/srv/unikeyx/new-api
```

Copy these files there:

```text
deploy/azure40/new-api/docker-compose.yml
deploy/azure40/new-api/.env.example -> .env
deploy/azure40/new-api/deploy.sh
```

Then create runtime directories:

```bash
mkdir -p /srv/unikeyx/new-api/data /srv/unikeyx/new-api/logs
```

## First deploy

```bash
cd /srv/unikeyx/new-api
cp .env.example .env
chmod +x deploy.sh
./deploy.sh latest
```

## Routine release path

1. Push to `main`.
2. Wait for `Publish GHCR Image` to finish.
3. Run `Deploy Azure40` with `image_tag=latest` or a `sha-*` tag.

## Rollback

Use the deploy workflow again with an older `sha-*` tag that already exists in GHCR.
