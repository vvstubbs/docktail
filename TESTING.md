# Testing DockTail Control Plane Sync

This guide explains how to manually verify the Control Plane Sync feature.

## Prerequisites

1. **Tailscale API Key:** Generate a key at [Tailscale Admin Console > Settings > Keys](https://login.tailscale.com/admin/settings/keys).
2. **Tailnet Name:** (Optional) Your tailnet domain (e.g., `example.com` or `user@gmail.com`).
3. **Docker & Tailscale:** Ensure Docker is running and Tailscale is installed/authenticated on your host.

## Setup

1. **Configure Environment:**
    Copy the example environment file:

    ```bash
    cp .env.example .env
    ```

    Edit `.env` and add your `TAILSCALE_API_KEY`.

2. **Review Test Configuration:**
    Check `docker-compose.test.yaml`. It defines:
    * `tailscale-sidecar`: A Tailscale container acting as the "host" daemon (for consistency if testing on macOS).
    * `docktail-test`: The DockTail instance built from source, connected to the sidecar.
    * `docktail-test-service`: An Nginx container labeled with:
        * Name: `svc:docktail-test-svc`
        * Tags: `tag:docktail-test`

## Running the Test

1. **Start the Stack:**

    ```bash
    docker compose -f docker-compose.test.yaml up --build
    ```

2. **Verify Logs:**
    Watch the logs for the `docktail-test` container. You should see:
    * `Configuration loaded` with `api_sync_enabled=true`.
    * `Syncing service definitions to Control Plane`.
    * `Successfully synced service definition to Control Plane` for `svc:docktail-test-svc`.

3. **Verify in Tailscale Console:**
    * Go to [Tailscale Admin Console > Services](https://login.tailscale.com/admin/services).
    * Look for a service named `docktail-test-svc`.
    * Verify it has the tag `tag:docktail-test`.

4. **Verify Control Plane Sync Logic (Debug Logs):**
    * Check the logs for `Fetching existing service definition`.
    * If the service already existed, look for `Found existing service addresses, preserving in update`.
    * If it was new, look for `Service does not exist, creating new (without addrs)`.
    * Look for `Sending Control Plane request` to see the exact JSON payload sent to the API.

## Cleanup

1. **Stop Containers:**

    ```bash
    docker-compose -f docker-compose.test.yaml down
    ```

2. **Note on Deletion:**
    DockTail uses a **Conservative Deletion Strategy**. Stopping the container will **NOT** remove the service definition from the Tailscale Admin Console. You must manually delete it from the console if you wish to clean it up completely.
