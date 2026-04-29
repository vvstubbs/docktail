# Agent Instructions

## Links

- [Tailscale Services Documentation](https://tailscale.com/kb/1552/tailscale-services)
- [Tailscale Service Configuration Reference](https://tailscale.com/kb/1589/tailscale-services-configuration-file)
- [Tailscale ACL Documentation](https://tailscale.com/kb/1337/policy-syntax)
- [Docker SDK for Go](https://docs.docker.com/engine/api/sdk/)

## Documentation Maintenance

When changing user-facing behavior, labels, environment variables, setup steps, examples, networking behavior, Tailscale permissions, supported protocols, or cleanup/reconciliation behavior, update the documentation in the same change.

The canonical documentation source is `docs/*.md`. Generated website documentation artifacts are build outputs and should not be committed as source-of-truth content.

Keep these aligned:

- `docs/*.md` for canonical human-facing documentation.
- `README.md` for the short overview, quick start, common examples, and links only.
- Generated website docs at `website/docs/index.html`.
- Generated Markdown and agent files at `website/docs.md`, `website/llms.txt`, and `website/llms-full.txt`.

After editing canonical docs, you may regenerate local ignored website artifacts with:

```bash
go run ./tools/docsgen
```

When adding a new feature, include the relevant docs page update and at least one concise example if the feature changes configuration.

## Project-Specific Constraints

- Do not run tests, start the software, start the dev server, start Docker Compose, or execute migrations unless explicitly asked by the developer.
- Do not read entire translation files. Make targeted reads and edits only.
- Do not add yourself as a co-author in commits.
