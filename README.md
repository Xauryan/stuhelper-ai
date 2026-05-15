# StuHelper AI

StuHelper AI is a self-hosted AI API gateway for local customization and production Docker deployment.

## Docker Image

Release images are published to GitHub Container Registry when a version tag is pushed.

```text
ghcr.io/xauryan/stuhelper-ai:<version>
ghcr.io/xauryan/stuhelper-ai:latest
```

`latest` points to the newest published version tag.

## Development

Backend code is written in Go. The product frontend defaults to `web/classic` and uses Bun.

```powershell
cd web/classic
bun install
bun run dev
```

For local container testing:

```powershell
docker compose -f docker-compose.dev.yml up -d --build
```

## Production Switch

On an existing Docker Compose deployment, replace the application image with:

```yaml
image: ghcr.io/xauryan/stuhelper-ai:latest
```

Then pull and restart the application service:

```bash
docker compose pull
docker compose up -d
```

## Release

Create and push a version tag:

```powershell
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions will build and publish the versioned image plus `latest`.

## License

This project is distributed under the GNU Affero General Public License v3.0. See `LICENSE`, `NOTICE`, and `THIRD-PARTY-LICENSES.md`.
