# ts-svc-autopilot

Automatically expose Docker containers as Tailscale Services using label-based configuration - zero-config service mesh for your homelab.

## What is Tailscale Services?

[Tailscale Services](https://tailscale.com/kb/1552/tailscale-services) (currently in beta) decouples internal resources from the devices hosting them by creating named services with stable DNS names. Instead of connecting to specific devices, users connect to services using MagicDNS names while Tailscale automatically routes traffic to available hosts.

**Key benefits:**
- **Stable addressing**: `http://myapp.tailnet-name.ts.net` stays the same even when moving between hosts
- **High availability**: Multiple hosts can advertise the same service
- **Access control**: Leverage Tailscale's ACLs and identity-based authentication
- **No public exposure**: Services only accessible within your tailnet

**Documentation:**
- [Tailscale Services Overview](https://tailscale.com/kb/1552/tailscale-services)
- [Service Configuration File Syntax](https://tailscale.com/kb/1589/tailscale-services-configuration-file)

## The Problem

Manually configuring Tailscale Services for Docker containers requires:
1. Creating service definitions in the admin console
2. Writing JSON configuration files
3. Running `tailscale serve set-config` commands
4. Running `tailscale serve advertise` commands
5. Cleaning up when containers stop
6. Updating configs when containers restart with new IPs

This is tedious, error-prone, and doesn't scale well for homelabs with many services.

## The Solution

**ts-svc-autopilot** watches your Docker daemon and automatically manages Tailscale Services based on container labels - similar to how Traefik discovers services.

**Just add labels to your containers:**
```yaml
services:
  nginx:
    image: nginx:latest
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=web"
      - "ts-svc.port=443"
      - "ts-svc.target=80"
      - "ts-svc.protocol=http"
```

**ts-svc-autopilot handles the rest:**
- Detects container starts/stops via Docker events
- Extracts bridge network IP addresses
- Generates Tailscale service configurations
- Applies configs and advertises services
- Gracefully drains and cleans up on container stop
- Updates configs when containers restart with new IPs

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

## How It Works

### 1. Container Discovery
Monitors Docker events API for container lifecycle events (start, stop, die, restart).

### 2. Label Parsing
Extracts Tailscale service configuration from container labels:
- `ts-svc.enable`: Enable autopilot for this container
- `ts-svc.service`: Service name (becomes `svc:name`)
- `ts-svc.port`: External port to expose on Tailscale
- `ts-svc.target`: Internal container port
- `ts-svc.protocol`: Protocol (`http`, `https`, `tcp`, `tls-terminated-tcp`)

### 3. IP Address Resolution
Queries Docker API for container's bridge network IP address (e.g., `172.17.0.5`).

### 4. Configuration Generation
Creates Tailscale service configuration JSON:
```json
{
  "version": "0.0.1",
  "services": {
    "svc:web": {
      "endpoints": {
        "tcp:443": "http://172.17.0.5:80"
      }
    }
  }
}
```

### 5. Configuration Application
Executes Tailscale CLI commands:
```bash
tailscale serve set-config --all /tmp/config.json
tailscale serve advertise svc:web
```

### 6. Stateless Operation
On startup and at regular intervals, autopilot:
1. Queries Docker for all running containers with `ts-svc.enable=true`
2. Reads current Tailscale service configuration via `tailscale serve get-config --all`
3. Reconciles differences and applies necessary changes
4. No persistent state storage required - truth comes from Docker API

## Prerequisites

**On the Docker host:**
1. Tailscale installed and authenticated
2. Host must have a tag-based identity (not user-based)
3. Docker daemon running with API socket exposed
4. (Recommended) Auto-approval ACL policies configured

**Auto-approval example** (`/admin/acls`):
```json
{
  "autoApprovers": {
    "services": {
      "tag:homelab-service": ["tag:homelab"]
    }
  }
}
```

This allows devices tagged `tag:homelab` to automatically advertise services tagged `tag:homelab-service`.

## Usage

### Docker Compose Deployment

```yaml
version: '3.8'

services:
  ts-svc-autopilot:
    image: ghcr.io/yourusername/ts-svc-autopilot:latest
    container_name: ts-svc-autopilot
    restart: unless-stopped
    volumes:
      # Docker socket for container monitoring
      - /var/run/docker.sock:/var/run/docker.sock:ro
      # Tailscale socket for CLI communication
      - /var/run/tailscale/tailscaled.sock:/var/run/tailscale/tailscaled.sock
    environment:
      - LOG_LEVEL=info
      - RECONCILE_INTERVAL=60s
    # Optional: network mode host to avoid networking issues
    # network_mode: host

  # Example application
  nginx:
    image: nginx:latest
    labels:
      - "ts-svc.enable=true"
      - "ts-svc.service=web"
      - "ts-svc.port=443"
      - "ts-svc.target=80"
      - "ts-svc.protocol=https"

  # Another example - database
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

### Running the Containers

```bash
# Start everything
docker compose up -d

# Check autopilot logs
docker logs -f ts-svc-autopilot

# Verify services are advertised
tailscale serve status --json

# Access from any device in your tailnet
curl https://web.your-tailnet.ts.net
psql postgresql://user@db.your-tailnet.ts.net:5432/database
```

### Label Reference

| Label | Required | Description | Example |
|-------|----------|-------------|---------|
| `ts-svc.enable` | Yes | Enable autopilot for container | `true` |
| `ts-svc.service` | Yes | Service name (alphanumeric, hyphens) | `web`, `api-v2` |
| `ts-svc.port` | Yes | Port exposed on Tailscale | `443`, `8080` |
| `ts-svc.target` | Yes | Internal container port | `80`, `3000` |
| `ts-svc.protocol` | Yes | Protocol type | `http`, `https`, `tcp` |

**Supported protocols:**
- `http`: Layer 7 HTTP forwarding
- `https`: Layer 7 HTTPS forwarding (auto TLS cert)
- `tcp`: Layer 4 TCP forwarding
- `tls-terminated-tcp`: Layer 4 with TLS termination

## Implementation Details

### Technology Stack
- **Language**: Go 1.21+
- **Docker SDK**: github.com/docker/docker/client
- **Logging**: zerolog or logrus
- **State Management**: Stateless - queries Docker API on-demand

### Key Components

**Event Listener**
```go
// Watches Docker events stream
eventsChan, errChan := dockerClient.Events(ctx, types.EventsOptions{
    Filters: filters.NewArgs(
        filters.Arg("type", "container"),
    ),
})
```

**Service Manager**
```go
// Handles service lifecycle (stateless)
type ServiceManager struct {
    dockerClient   *docker.Client
    tsConfig       *TailscaleConfig
}

// Reconciles by querying current state
func (sm *ServiceManager) Reconcile() error {
    // Query Docker for all containers with ts-svc.enable=true
    // Query Tailscale for current config
    // Apply differences
}
```

**Configuration Builder**
```go
// Generates Tailscale service JSON
func buildServiceConfig(containers []*Container) *TailscaleServiceConfig {
    config := &TailscaleServiceConfig{
        Version:  "0.0.1",
        Services: make(map[string]ServiceDefinition),
    }
    // ... build logic from container labels
}
```

### Error Handling

- **Docker daemon disconnects**: Reconnect with exponential backoff
- **Container IP not available**: Skip container, retry on next reconciliation
- **Tailscale CLI failures**: Log error, retry on next reconciliation
- **Invalid labels**: Log warning, skip container

### State Reconciliation

Periodic reconciliation loop (every 60s by default):
1. Query all running containers with `ts-svc.enable=true`
2. Query current Tailscale service configuration
3. Calculate diff (containers added/removed/changed)
4. Apply only necessary changes (drain → clear → configure → advertise)

On Docker events (start/stop/die/restart):
1. Trigger immediate reconciliation
2. No separate event-based state tracking needed

## Advanced Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `RECONCILE_INTERVAL` | `60s` | State reconciliation interval |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon socket |
| `TAILSCALE_SOCKET` | `/var/run/tailscale/tailscaled.sock` | Tailscale daemon socket |

### Multiple Networks

For containers on multiple Docker networks:
```yaml
labels:
  - "ts-svc.enable=true"
  - "ts-svc.service=app"
  - "ts-svc.network=custom_bridge"  # Specify which network
  - "ts-svc.port=443"
  - "ts-svc.target=8080"
  - "ts-svc.protocol=https"
```

### High Availability

Multiple containers can advertise the same service name:
```yaml
services:
  web-01:
    image: nginx
    labels:
      - "ts-svc.service=web"  # Same name
      
  web-02:
    image: nginx
    labels:
      - "ts-svc.service=web"  # Same name
```

Tailscale automatically load balances across available hosts.

## Limitations

- **TCP only**: Tailscale Services currently only support TCP protocol
- **Bridge networks**: Container must be on a Docker network the host can route to
- **Linux recommended**: Layer 3 endpoints only work on Linux
- **Tag-based identity**: Host machine must use Tailscale tags, not user authentication
- **No hairpinning**: Containers cannot access their own Tailscale service endpoint

## Links

- [Tailscale Services Documentation](https://tailscale.com/kb/1552/tailscale-services)
- [Tailscale Service Configuration Reference](https://tailscale.com/kb/1589/tailscale-services-configuration-file)
- [Tailscale ACL Documentation](https://tailscale.com/kb/1337/policy-syntax)
- [Docker SDK for Go](https://docs.docker.com/engine/api/sdk/)