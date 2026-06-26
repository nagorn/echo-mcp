# Contributing

Echo MCP is source-available and maintained primarily by the author.

Feedback, documentation fixes, and small bug fixes are welcome. Substantial
feature work should start with an issue or design discussion before code is
written.

## Before Proposing a Change

Read:

- [README](README.md)
- [Architecture decisions](docs/adr/)
- [Developer Usage Guide](docs/guides/developer-usage.md)
- [Security policy](SECURITY.md)

Keep changes aligned with the existing architecture:

- Preserve the MCP Control Plane and HTTP Data Plane separation.
- Keep application code unaware of Echo MCP.
- Do not add new product surfaces without an approved design.
- Do not add authentication, persistence, UI, CLI, metrics, audit logs, or
  production deployment behavior as incidental changes.
- Do not make Echo MCP generate behavior from OpenAPI or AI intent.

## Verification

Before submitting a change, run:

```bash
go test ./...
go test -race ./...
go vet ./...
go build -o bin/echo-mcp ./cmd/echo-mcp
make test
make build
git diff --check
```

If Docker behavior is relevant and Docker is available:

```bash
make docker-build
```

## License

Echo MCP is source-available under the Elastic License 2.0. It is not OSI open
source.

Echo MCP is free to use for internal development and testing under the license
terms. Commercial or enterprise licensing may be offered separately later.

By contributing, you agree that your contribution may be distributed under the
project's license, the Elastic License 2.0.

Do not submit code or documentation that you do not have the right to contribute.
