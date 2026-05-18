# Docker Deployment

This guide describes how to run `go-crypto-arb` with Docker. The container runs the backend API by default and exposes:

- `GET /health`
- protected `/api/v1/*` routes
- `GET /metrics` when Prometheus metrics are enabled in config

The application is still monitoring-only in Docker. It does not place orders or execute trades.

## Prerequisites

- Docker Engine with BuildKit support.
- Docker Compose v2 for the recommended flow.
- A configured `.env` file.
- A configured `configs/config.yaml` file.

Create local runtime files:

```bash
cp .env.example .env
cp configs/config.example.yaml configs/config.yaml
```

Set a real API key in `.env`:

```env
API_KEY=change-me
CONFIG_PATH=./configs/config.yaml
HTTP_ADDR=:8080
```

For Docker, `CONFIG_PATH=./configs/config.yaml` resolves inside the container as `/app/configs/config.yaml` because the image uses `/app` as its working directory.

Enable Prometheus metrics in `configs/config.yaml` if you want `/metrics`:

```yaml
metrics:
  prometheus_enabled: true
  prometheus_path: /metrics
```

## Recommended: Docker Compose

Use the included example compose file as a starting point:

```bash
cp docker-compose.example.yml docker-compose.yml
docker compose up --build -d
```

The compose service:

- builds the local `Dockerfile`
- loads `.env`
- publishes container port `8080` to host port `8080`
- mounts `./configs/config.yaml` read-only into the container
- restarts unless stopped

Check service status:

```bash
docker compose ps
docker compose logs -f api
```

Check the API:

```bash
curl http://localhost:8080/health
curl -H "X-API-Key: change-me" http://localhost:8080/api/v1/snapshot
curl http://localhost:8080/metrics
```

Stop the deployment:

```bash
docker compose down
```

## Plain Docker Run

Build the image:

```bash
docker build -t go-crypto-arb:local .
```

Run the API container:

```bash
docker run -d \
  --name go-crypto-arb-api \
  --env-file .env \
  -p 8080:8080 \
  -v "$PWD/configs/config.yaml:/app/configs/config.yaml:ro" \
  --restart unless-stopped \
  go-crypto-arb:local
```

Inspect logs:

```bash
docker logs -f go-crypto-arb-api
```

Stop and remove the container:

```bash
docker stop go-crypto-arb-api
docker rm go-crypto-arb-api
```

## Prometheus Scraping

The root [prometheus.yml](../prometheus.yml) file scrapes the API every `5s`:

```yaml
global:
  scrape_interval: 5s
  evaluation_interval: 5s

scrape_configs:
  - job_name: go-crypto-arb
    metrics_path: /metrics
    static_configs:
      - targets:
          - localhost:8080
```

If Prometheus runs directly on the host and the API container publishes `8080:8080`, `localhost:8080` works.

If Prometheus runs in another Docker container on the same compose network, use the service name instead:

```yaml
static_configs:
  - targets:
      - go-crypto-arb-api:8080
```

## Running the TUI Container

The image also includes `go-crypto-arb-tui`, but the default command runs the API. You can start the TUI manually against a running backend:

```bash
docker run --rm -it \
  --env-file .env \
  -e CONFIG_PATH=./configs/config.yaml \
  -v "$PWD/configs/config.yaml:/app/configs/config.yaml:ro" \
  go-crypto-arb:local \
  go-crypto-arb-tui
```

For a TUI container to reach an API container, set `tui.backend_url` in `configs/config.yaml` to a reachable URL. In Compose, that usually means `http://api:8080`; from the host, `http://localhost:8080` is usually enough.

## IBKR Notes

IBKR support is market-data only and partial. If the API container needs to reach TWS or IB Gateway on the host, set the IBKR host in `.env` or `configs/config.yaml` to an address reachable from the container.

Common options:

- Linux Docker host: use the Docker bridge gateway address, often `172.17.0.1`.
- Docker Desktop: use `host.docker.internal`.
- Same Compose network: run the gateway in another service and use that service name.

Keep `trading_enabled: false`. Config validation treats IBKR trading as a hard error.

## Operational Checks

After deployment, check:

```bash
curl http://localhost:8080/health
curl -H "X-API-Key: change-me" http://localhost:8080/api/v1/providers/health
curl http://localhost:8080/metrics
```

Useful maintenance commands:

```bash
docker compose pull
docker compose up --build -d
docker compose logs --tail=200 api
docker compose restart api
```

## Troubleshooting

- `401 Unauthorized`: the `X-API-Key` header is missing or does not match `API_KEY`.
- `404` on `/metrics`: `metrics.prometheus_enabled` is disabled or `metrics.prometheus_path` is different.
- Config file not found: make sure `configs/config.yaml` exists and is mounted to `/app/configs/config.yaml`.
- No market data: inspect provider health with `/api/v1/providers/health` and container logs.
- IBKR unreachable: verify the host/port is reachable from inside the container, not only from the host.
