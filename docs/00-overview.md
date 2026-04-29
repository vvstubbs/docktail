# DockTail Documentation

DockTail exposes Docker containers as Tailscale Services using label-based configuration. It watches Docker events, reads `docktail.*` labels, and advertises matching containers through the local Tailscale daemon.

## What DockTail Does

- Discovers labeled Docker containers automatically.
- Proxies directly to container IPs by default, so app containers do not need published Docker ports.
- Advertises HTTP, HTTPS, TCP, and TLS-terminated TCP services through Tailscale.
- Supports Tailscale HTTPS with automatic certificates.
- Supports Tailscale Funnel for public internet access.
- Supports multiple Tailscale services from one container.
- Reconciles state when containers restart and container IPs change.
- Runs as a stateless Docker container.

## Recommended Reading Order

1. Start with [Quick Start](#quick-start) for a minimal Compose setup.
2. Read [Installation](#installation) for host Tailscale and sidecar options.
3. Configure Tailscale permissions in [Tailscale Admin Setup](#tailscale-admin-setup).
4. Use [Labels](#labels) and [Examples](#examples) when exposing real services.
5. Check [Reference](#reference) for all labels, environment variables, protocols, and behavior notes.
