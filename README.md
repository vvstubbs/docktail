# ts-svc-autopilot

Automatically expose Docker containers as Tailscale Services using label-based configuration - zero-config service mesh for your homelab.

## Quick Start

### Prerequisites

1. Tailscale installed and authenticated on your Docker host
2. Host must have a tag-based identity (not user-based)
3. Docker daemon running with API socket exposed
4. (Recommended) Auto-approval ACL policies configured in your Tailscale admin console

### Installation

1. Clone this repository:
```bash
git clone https://github.com/marvinvr/ts-svc-autopilot
cd ts-svc-autopilot
```

2. Start the services:
```bash
docker compose up -d
```

3. Check the logs:
```bash
docker logs -f ts-svc-autopilot
```

4. Verify services are advertised:
```bash
tailscale serve status --json
```

### Usage

**🚨 CRITICAL:** Container ports MUST be published to host. Tailscale serve only supports `localhost` proxies.

```yaml
services:
  myapp:
    image: nginx:latest
    ports:
      - "9080:80"  # REQUIRED! HOST:CONTAINER format
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=myapp"
      - "ts-svc.port=443"                  # Port on Tailscale (default: 80)
      - "ts-svc.target=80"                 # CONTAINER port (RIGHT side of "9080:80")
      - "ts-svc.target-protocol=https"     # Protocol (default: http)
```

**Port Mapping Rules:**
- `ports:` = `"HOST:CONTAINER"` (e.g., `"9080:80"` = host 9080 → container 80)
- `ts-svc.target` = CONTAINER port (always the RIGHT side)
- Result: Tailscale:443 → localhost:9080 → Container:80

Access from any device in your tailnet:
```bash
curl https://myapp.your-tailnet.ts.net
```

### Available Labels

| Label | Required | Default | Description | Example |
|-------|----------|---------|-------------|---------|
| `ts-svc.enable` | Yes | - | Enable autopilot for container | `true` |
| `ts-svc.service` | Yes | - | Service name | `web`, `api-v2` |
| `ts-svc.target` | Yes | - | **CONTAINER** port (RIGHT side of `ports:`) | `80`, `3000` |
| `ts-svc.port` | No | `80` | Port exposed on Tailscale | `443`, `8080` |
| `ts-svc.target-protocol` | No | `http` | Protocol type | `http`, `https`, `tcp` |

**Critical:** If `ports: "9080:80"`, then `ts-svc.target=80` (container port, NOT 9080)

### Supported Protocols

- `http`: Layer 7 HTTP forwarding
- `https`: Layer 7 HTTPS forwarding (auto TLS cert)
- `tcp`: Layer 4 TCP forwarding
- `tls-terminated-tcp`: Layer 4 with TLS termination

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `RECONCILE_INTERVAL` | `60s` | State reconciliation interval |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon socket |
| `TAILSCALE_SOCKET` | `/var/run/tailscale/tailscaled.sock` | Tailscale daemon socket |

## How It Works

1. **Container Discovery**: Monitors Docker events API for container lifecycle events (start, stop, die, restart)
2. **Label Parsing**: Extracts Tailscale service configuration from container labels
3. **Port Detection**: Queries Docker API for published host ports
4. **Configuration Generation**: Creates Tailscale service configuration proxying to `localhost:HOST_PORT`
5. **Configuration Application**: Executes Tailscale CLI commands to apply config and advertise services
6. **Stateless Operation**: Periodically reconciles state by querying Docker and Tailscale APIs

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     Docker Host                          │
│                                                          │
│  ┌──────────────────┐         ┌──────────────────┐     │
│  │  ts-svc-autopilot│────────▶│ Tailscale Daemon │     │
│  │   (Container)    │  CLI    │   (Host Process) │     │
│  └────────┬─────────┘         └────────┬─────────┘     │
│           │                             │                │
│           │ Docker Socket               │ Proxies to    │
│           │ Monitoring                  │ localhost     │
│           ▼                             ▼                │
│  ┌──────────────────┐         ┌──────────────────┐     │
│  │   App Container  │◀────────│  localhost:9080  │     │
│  │   Port 80        │  Mapped │  localhost:9081  │     │
│  │  ports: 9080:80  │◀────────│                  │     │
│  └──────────────────┘         └──────────────────┘     │
│                                                          │
└─────────────────────────────────────────────────────────┘
                         │
                         │ Tailscale Network
                         ▼
              ┌────────────────────┐
              │  Tailnet Clients   │
              │  Access services:  │
              │  web.tailnet.ts.net│
              └────────────────────┘

Flow: Tailscale → localhost:9080 → Container:80
```

## Examples

### Web Application

```yaml
services:
  nginx:
    image: nginx:latest
    ports:
      - "8080:80"  # HOST:CONTAINER
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=web"
      - "ts-svc.port=443"              # Tailscale port (default: 80)
      - "ts-svc.target=80"             # CONTAINER port (right side)
      - "ts-svc.target-protocol=https" # Protocol (default: http)
```

### Database

```yaml
services:
  postgres:
    image: postgres:16
    ports:
      - "5432:5432"  # HOST:CONTAINER (same on both sides)
    environment:
      POSTGRES_PASSWORD: secret
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=db"
      - "ts-svc.port=5432"           # Tailscale port (default: 80)
      - "ts-svc.target=5432"         # CONTAINER port
      - "ts-svc.target-protocol=tcp" # Protocol (default: http)
```

### API (Different Host and Container Ports)

```yaml
services:
  api:
    image: myapi:latest
    ports:
      - "8080:3000"  # HOST:CONTAINER - Host 8080 → Container 3000
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=api"
      - "ts-svc.port=443"              # Tailscale port (default: 80)
      - "ts-svc.target=3000"           # CONTAINER port (right side: "8080:3000")
      - "ts-svc.target-protocol=http"  # Protocol (default: http)
```

## Building from Source

```bash
# Build binary
go build -o ts-svc-autopilot .

# Build Docker image
docker build -t ts-svc-autopilot:latest .

# Run locally
./ts-svc-autopilot
```

## Documentation

See [CLAUDE.md](CLAUDE.md) for detailed project documentation including:
- Tailscale Services overview
- Complete architecture details
- Implementation details
- Advanced configuration
- Troubleshooting

## Links

- [Tailscale Services Documentation](https://tailscale.com/kb/1552/tailscale-services)
- [Tailscale Service Configuration Reference](https://tailscale.com/kb/1589/tailscale-services-configuration-file)
- [Docker SDK for Go](https://docs.docker.com/engine/api/sdk/)

## License

MIT
