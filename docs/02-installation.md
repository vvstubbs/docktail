## Installation

DockTail needs access to the Docker socket and a Tailscale socket. Use the host setup when Tailscale already runs on the Docker host. Use the sidecar setup when the host should not install Tailscale directly.

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

The `--reset` flag briefly drops the Tailscale connection. If you are connected through SSH over Tailscale, your session may be interrupted until Tailscale reconnects.

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

Set `TAILSCALE_AUTH_KEY` to authenticate the Tailscale container. Generate it in the Tailscale Admin Console under Settings -> Keys. The sidecar should advertise `tag:server` so it can satisfy the ACL auto-approver example below.
