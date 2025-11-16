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

Add labels to any Docker container to expose it via Tailscale Services:

```yaml
services:
  myapp:
    image: nginx:latest
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=myapp"
      - "ts-svc.port=443"
      - "ts-svc.target=80"
      - "ts-svc.protocol=https"
```

Access from any device in your tailnet:
```bash
curl https://myapp.your-tailnet.ts.net
```

### Available Labels

| Label | Required | Description | Example |
|-------|----------|-------------|---------|
| `ts-svc.enable` | Yes | Enable autopilot for container | `true` |
| `ts-svc.service` | Yes | Service name | `web`, `api-v2` |
| `ts-svc.port` | Yes | Port exposed on Tailscale | `443`, `8080` |
| `ts-svc.target` | Yes | Internal container port | `80`, `3000` |
| `ts-svc.protocol` | Yes | Protocol type | `http`, `https`, `tcp` |
| `ts-svc.network` | No | Specific Docker network | `custom_bridge` |

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
3. **IP Address Resolution**: Queries Docker API for container's bridge network IP address
4. **Configuration Generation**: Creates Tailscale service configuration JSON
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
│  └────────┬─────────┘         └──────────────────┘     │
│           │                                              │
│           │ Docker Socket                                │
│           │ Monitoring                                   │
│           ▼                                              │
│  ┌──────────────────┐         ┌──────────────────┐     │
│  │   App Container  │         │  App Container   │     │
│  │  172.17.0.5:80   │         │  172.17.0.6:3000 │     │
│  │  labels: {...}   │         │  labels: {...}   │     │
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
```

## Examples

### Web Application

```yaml
services:
  nginx:
    image: nginx:latest
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=web"
      - "ts-svc.port=443"
      - "ts-svc.target=80"
      - "ts-svc.protocol=https"
```

### Database

```yaml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: secret
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=db"
      - "ts-svc.port=5432"
      - "ts-svc.target=5432"
      - "ts-svc.protocol=tcp"
```

### API with Custom Network

```yaml
services:
  api:
    image: myapi:latest
    networks:
      - custom
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=api"
      - "ts-svc.port=8080"
      - "ts-svc.target=3000"
      - "ts-svc.protocol=http"
      - "ts-svc.network=custom"

networks:
  custom:
    driver: bridge
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
