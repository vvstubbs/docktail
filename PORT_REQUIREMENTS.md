# Port Publishing Requirements - Quick Reference

## TL;DR

**Container ports MUST be published to the host.** Tailscale serve only supports `localhost` proxies.

## Correct Configuration

```yaml
services:
  myapp:
    image: nginx
    ports:
      - "9080:80"  # HOST:CONTAINER ← REQUIRED!
    labels:
      - "ts-svc.target=80"  # ← CONTAINER port (RIGHT side: "9080:80")
```

## Port Label Rules

| Component | Format | Example | Explanation |
|-----------|--------|---------|-------------|
| `ports:` | `"HOST:CONTAINER"` | `"9080:80"` | Host port 9080 maps to container port 80 |
| `ts-svc.target` | CONTAINER port | `80` | RIGHT side of the `ports:` mapping |
| **Autopilot detects** | HOST port automatically | `9080` | Auto-detected from port bindings |
| **Result** | Tailscale → localhost → Container | `443 → localhost:9080 → 80` | Complete proxy chain |

## Common Mistakes

### ❌ Wrong - Using Host Port in Target

```yaml
ports:
  - "9080:80"
labels:
  - "ts-svc.target=9080"  # ❌ WRONG! This is the HOST port
```

### ✅ Correct - Using Container Port in Target

```yaml
ports:
  - "9080:80"
labels:
  - "ts-svc.target=80"  # ✅ CORRECT! This is the CONTAINER port
```

### ❌ Wrong - Missing Port Publishing

```yaml
# No ports: section
labels:
  - "ts-svc.target=80"  # ❌ WRONG! Port must be published
```

### ✅ Correct - With Port Publishing

```yaml
ports:
  - "9080:80"  # ✅ CORRECT! Port is published
labels:
  - "ts-svc.target=80"
```

## Why This Matters

Tailscale's `serve` command has a limitation:
- **Only supports proxying to `localhost` or `127.0.0.1`**
- **Cannot proxy directly to Docker container IPs** (like `172.17.0.5`)

Therefore:
1. Container ports MUST be published to the host
2. Autopilot proxies to `localhost:HOST_PORT`
3. Docker routes `localhost:HOST_PORT` → Container:CONTAINER_PORT

## Examples

### Example 1: Same Port on Both Sides

```yaml
services:
  db:
    image: postgres:16
    ports:
      - "5432:5432"  # HOST:CONTAINER - Same port
    labels:
      - "ts-svc.port=5432"    # Tailscale port
      - "ts-svc.target=5432"  # Container port (same as host port)

# Flow: Tailscale:5432 → localhost:5432 → Container:5432
```

### Example 2: Different Ports

```yaml
services:
  web:
    image: nginx
    ports:
      - "9080:80"  # HOST:CONTAINER - Different ports
    labels:
      - "ts-svc.port=443"   # Tailscale port
      - "ts-svc.target=80"  # Container port (NOT 9080!)

# Flow: Tailscale:443 → localhost:9080 → Container:80
```

### Example 3: Multiple Containers

```yaml
services:
  web1:
    image: nginx
    ports:
      - "9080:80"  # Host 9080 → Container 80
    labels:
      - "ts-svc.service=web1"
      - "ts-svc.port=443"
      - "ts-svc.target=80"  # Container port

  web2:
    image: nginx
    ports:
      - "9081:80"  # Host 9081 → Container 80 (different host port!)
    labels:
      - "ts-svc.service=web2"
      - "ts-svc.port=8443"
      - "ts-svc.target=80"  # Same container port, different host port

# Flow:
# - Tailscale:443  → localhost:9080 → web1:80
# - Tailscale:8443 → localhost:9081 → web2:80
```

## Error Messages

If you see this error:
```
container port 80 is NOT published to host
```

**Fix:** Add `ports: ["80:80"]` (or any `"HOST:80"` mapping) to your docker-compose.yaml

## Quick Checklist

- [ ] Container has `ports:` section in docker-compose.yaml
- [ ] Port mapping uses `"HOST:CONTAINER"` format
- [ ] `ts-svc.target` matches the CONTAINER port (RIGHT side)
- [ ] Host port doesn't conflict with other services
