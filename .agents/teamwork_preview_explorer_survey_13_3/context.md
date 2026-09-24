# Survey Explorer 3 Context: Infrastructure Gateway & Compose Modularization

Mission:
Survey `docker-compose.yml` and `docker/Caddyfile` in `pegasus.x`:
- Inspect `docker-compose.yml` to see all current services, networks, volumes, ports, and environment variables.
- Determine how to cleanly decompose `docker-compose.yml` into a base definition (`docker-compose.base.yml` or `docker-compose.yml`) with environment overlays (`docker-compose.dev.yml`, `docker-compose.prod.yml`).
- Inspect `docker/Caddyfile` (or `Caddyfile`) to map all domain route blocks, reverse proxy rules, TAS-IX direct peering, and TLS termination configs.
- Determine how to decompose the Caddyfile into modular domain snippets (`caddy.d/api.caddy`, `caddy.d/ws.caddy`, `caddy.d/portal.caddy`, etc.) while ensuring syntax validity and zero route loss.
- Check current docker compose config validation commands and syntax checks.
- Propose a clean modularization and migration plan.
- Write your comprehensive findings to `handoff.md` in your working directory.
