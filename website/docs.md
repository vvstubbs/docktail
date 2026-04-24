# DockTail Documentation

DockTail automatically exposes Docker containers as Tailscale Services using label-based configuration.

Source: https://github.com/marvinvr/docktail

## Quick Start

Add DockTail to your Docker Compose file alongside your services:

```yaml
services:
  docktail:
    image: ghcr.io/marvinvr/docktail:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /var/run/tailscale:/var/run/tailscale
    environment:
      # Optional but recommended - enables auto-service-creation
      - TAILSCALE_OAUTH_CLIENT_ID=${TAILSCALE_OAUTH_CLIENT_ID}
      - TAILSCALE_OAUTH_CLIENT_SECRET=${TAILSCALE_OAUTH_CLIENT_SECRET}

  myapp:
    image: nginx:latest
    # No ports needed! DockTail proxies directly to container IP
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=myapp"
      - "docktail.service.port=80"
```

```bash
docker compose up -d
curl http://myapp.your-tailnet.ts.net
```

## Installation

### Tailscale On Host

Use this setup when Tailscale is already installed on the Docker host:

```yaml
services:
  docktail:
    image: ghcr.io/marvinvr/docktail:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /var/run/tailscale:/var/run/tailscale
    environment:
      - TAILSCALE_OAUTH_CLIENT_ID=${TAILSCALE_OAUTH_CLIENT_ID}
      - TAILSCALE_OAUTH_CLIENT_SECRET=${TAILSCALE_OAUTH_CLIENT_SECRET}
```

Mount `/var/run/tailscale` as a directory rather than mounting the socket file directly. When `tailscaled` restarts, it recreates the socket with a new inode; a directory mount stays in sync.

The host machine must advertise a tag that matches your ACL auto-approvers:

```bash
sudo tailscale up --advertise-tags=tag:server --reset
```

### Tailscale Sidecar

Use this setup when the host does not run Tailscale directly:

```yaml
services:
  tailscale:
    image: tailscale/tailscale:latest
    hostname: docktail-host
    environment:
      - TS_AUTHKEY=${TAILSCALE_AUTH_KEY}
      - TS_EXTRA_ARGS=--advertise-tags=tag:server
      - TS_STATE_DIR=/var/lib/tailscale
      - TS_SOCKET=/var/run/tailscale/tailscaled.sock
    volumes:
      - tailscale-state:/var/lib/tailscale
      - tailscale-socket:/var/run/tailscale
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    network_mode: host
    restart: unless-stopped

  docktail:
    image: ghcr.io/marvinvr/docktail:latest
    depends_on:
      - tailscale
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - tailscale-socket:/var/run/tailscale
    environment:
      - TAILSCALE_OAUTH_CLIENT_ID=${TAILSCALE_OAUTH_CLIENT_ID}
      - TAILSCALE_OAUTH_CLIENT_SECRET=${TAILSCALE_OAUTH_CLIENT_SECRET}

volumes:
  tailscale-state:
  tailscale-socket:
```

Set `TAILSCALE_AUTH_KEY` to authenticate the Tailscale container. The sidecar should advertise `tag:server` so it can satisfy the ACL auto-approver.

## Configuration

DockTail supports three operating modes.

### OAuth Credentials

Recommended. OAuth lets DockTail auto-create services in the Tailscale Admin Console.

Required OAuth permissions:

- General -> Services: Write
- Devices -> Core: Write
- Keys -> Auth Keys: Write, only when using the sidecar method

```yaml
environment:
  - TAILSCALE_OAUTH_CLIENT_ID=your-client-id
  - TAILSCALE_OAUTH_CLIENT_SECRET=your-client-secret
```

### API Key

Also auto-creates services, but Tailscale API keys expire.

```yaml
environment:
  - TAILSCALE_API_KEY=tskey-api-...
```

### Manual Mode

DockTail can run without credentials. It advertises services locally through the Tailscale CLI, but you must manually create service definitions in the Tailscale Admin Console and configure ACL auto-approvers.

## Admin Setup

Services need tag definitions in `tagOwners` and auto-approvers that allow the host to advertise services.

```json
{
  "tagOwners": {
    "tag:server": ["autogroup:admin"],
    "tag:container": ["tag:server"]
  },
  "autoApprovers": {
    "services": {
      "tag:container": ["tag:server"]
    }
  }
}
```

`tag:server` is assigned to the host machine or sidecar auth key. `tag:container` is the default tag DockTail assigns to services.

The first time a new service is advertised, it may need approval in the Tailscale Admin Console Services tab.

## Labeling Containers

DockTail watches for containers with `docktail.*` labels. Each labeled container becomes its own Tailscale service.

### Direct Container IP Proxying

By default, DockTail proxies directly to container IPs on the Docker bridge network. No port publishing is required.

```yaml
services:
  myapp:
    image: nginx:latest
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=myapp"
      - "docktail.service.port=80"
```

Set `docktail.service.direct=false` to use published port bindings instead.

### Service Labels

| Label | Required | Default | Description |
| --- | --- | --- | --- |
| `docktail.service.enable` | Yes | - | Enable DockTail for the container |
| `docktail.service.name` | Yes | - | Service name, such as `web` or `api` |
| `docktail.service.port` | Yes | - | Container port to proxy to |
| `docktail.service.direct` | No | `true` | Proxy directly to container IP |
| `docktail.service.network` | No | `bridge` | Docker network for container IP |
| `docktail.service.protocol` | No | Smart | Container protocol |
| `docktail.service.service-port` | No | Smart | Port Tailscale listens on |
| `docktail.service.service-protocol` | No | Smart | Tailscale-facing protocol |
| `docktail.tags` | No | `tag:container` | Comma-separated ACL tags |

Smart defaults:

- `protocol`: `https` if container port is `443`, otherwise `http`
- `service-port`: `443` if service protocol is `https`, otherwise `80`
- `service-protocol`: `https` if service port is `443`, matches `protocol` for TCP, otherwise `http`

### Funnel Labels

Funnel exposes a service to the public internet.

| Label | Required | Default | Description |
| --- | --- | --- | --- |
| `docktail.funnel.enable` | Yes | `false` | Enable Tailscale Funnel |
| `docktail.funnel.port` | Yes | - | Container port |
| `docktail.funnel.funnel-port` | No | `443` | Public port: `443`, `8443`, or `10000` |
| `docktail.funnel.protocol` | No | `https` | Protocol: `https`, `tcp`, or `tls-terminated-tcp` |

Notes:

- Only one funnel per port is supported by Tailscale.
- Funnel uses the machine hostname, not the service name: `https://<machine>.<tailnet>.ts.net`.
- Funnel-only containers can omit `docktail.service.enable`.

## Examples

### Web Application

```yaml
services:
  nginx:
    image: nginx:latest
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=web"
      - "docktail.service.port=80"
```

### HTTPS With Auto TLS

```yaml
services:
  api:
    image: myapi:latest
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=api"
      - "docktail.service.port=3000"
      - "docktail.service.service-port=443"
```

Access the service at `https://api.your-tailnet.ts.net`.

### Database Over TCP

```yaml
services:
  postgres:
    image: postgres:16
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=db"
      - "docktail.service.port=5432"
      - "docktail.service.protocol=tcp"
      - "docktail.service.service-port=5432"
```

### Multiple Services From One Container

Use numbered labels with `docktail.service.N.*`:

```yaml
services:
  gluetun:
    image: gluetun:latest
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=qbittorrent"
      - "docktail.service.port=8000"
      - "docktail.service.1.name=bitmagnet"
      - "docktail.service.1.port=8001"
```

Per-index overridable labels are `name`, `port`, `service-port`, `protocol`, and `service-protocol`.

### Custom Docker Network

```yaml
services:
  app:
    image: myapp:latest
    networks:
      - backend
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=app"
      - "docktail.service.port=3000"
      - "docktail.service.network=backend"

networks:
  backend:
```

### Legacy Published-Port Mode

```yaml
services:
  app:
    image: myapp:latest
    ports:
      - "8080:3000"
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=app"
      - "docktail.service.port=3000"
      - "docktail.service.direct=false"
```

### Public Website With Funnel

```yaml
services:
  website:
    image: nginx:latest
    labels:
      - "docktail.service.enable=true"
      - "docktail.service.name=website"
      - "docktail.service.port=80"
      - "docktail.service.service-port=443"
      - "docktail.funnel.enable=true"
      - "docktail.funnel.port=80"
```

Tailnet URL: `https://website.your-tailnet.ts.net`

Public Funnel URL: `https://your-machine.your-tailnet.ts.net`

## Reference

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `TAILSCALE_OAUTH_CLIENT_ID` | - | OAuth Client ID |
| `TAILSCALE_OAUTH_CLIENT_SECRET` | - | OAuth Client Secret |
| `TAILSCALE_API_KEY` | - | API key alternative to OAuth |
| `TAILSCALE_TAILNET` | - | Tailnet ID, defaults to key tailnet |
| `DEFAULT_SERVICE_TAGS` | `tag:container` | Default service tags |
| `IGNORE_SERVICE_NAMES` | - | Comma-separated service names DockTail must not drain or clear during reconciliation or shutdown cleanup |
| `LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |
| `RECONCILE_INTERVAL` | `60s` | State reconciliation interval |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon socket |
| `TAILSCALE_SOCKET` | `/var/run/tailscale/tailscaled.sock` | Tailscale daemon socket |

If both OAuth and API key are set, OAuth takes precedence.

`IGNORE_SERVICE_NAMES` accepts either bare names like `grafana` or fully qualified names like `svc:grafana`.

### Supported Protocols

Tailscale-facing `service-protocol` values:

- `http`: Layer 7 HTTP
- `https`: Layer 7 HTTPS with automatic TLS
- `tcp`: Layer 4 TCP
- `tls-terminated-tcp`: Layer 4 with TLS termination

Container-facing `protocol` values:

- `http`: HTTP backend
- `https`: HTTPS backend with a valid certificate
- `https+insecure`: HTTPS backend with a self-signed certificate
- `tcp`: TCP backend
- `tls-terminated-tcp`: TCP backend with TLS termination

## How It Works

1. DockTail monitors Docker events for container starts and stops.
2. DockTail extracts service configuration from container labels.
3. DockTail detects the container IP from Docker network settings.
4. DockTail creates Tailscale service config proxying to the container IP.
5. DockTail executes the Tailscale CLI to advertise services.
6. If OAuth or API key credentials are configured, DockTail creates service definitions through the Tailscale API.
7. DockTail periodically reconciles state and updates service config when container IPs change.

DockTail does not delete service definitions from the API when containers stop. This is a conservative deletion strategy.
