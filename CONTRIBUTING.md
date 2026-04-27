# Contributing

Thank you for improving Ollama Switchboard. This project is a local infrastructure tool, so contributions should be careful, testable, and respectful of users who rely on a stable local Ollama-compatible endpoint.

## Project Goals

Ollama Switchboard aims to provide:

- a stable local gateway for Ollama-compatible clients;
- unlimited local Ollama usage through local-first routing;
- controlled cloud fallback when local routing is not enough;
- predictable retry, cooldown, and failover behavior;
- clear configuration with safe defaults;
- practical tooling for developers, agents, and automation workflows.

Changes should move the project toward those goals without making local usage harder, less secure, or less observable.

## Ways to Contribute

Useful contributions include:

- bug fixes with focused regression tests;
- routing, failover, and streaming improvements;
- CLI and admin API usability improvements;
- security hardening for auth, secret handling, and network exposure;
- documentation improvements and examples;
- platform-specific setup notes;
- small refactors that reduce complexity without changing behavior.

Before starting a large change, open an issue or draft proposal so the design can be discussed early.

## Development Setup

Requirements:

- Go 1.23 or newer.
- Git.
- A local Ollama-compatible daemon for manual end-to-end testing.

Clone and verify:

```bash
git clone https://github.com/0PeterAdel/ollama-switchboard.git
cd ollama-switchboard
go test ./...
go build ./cmd/osb
```

Run locally:

```bash
go run ./cmd/osb setup --yes
go run ./cmd/osb serve
```

In another shell:

```bash
go run ./cmd/osb doctor
go run ./cmd/osb chat --model llama3 "hello"
```

## Branch and Commit Workflow

Use focused branches:

```bash
git switch -c fix/router-fallback
```

Keep commits intentional. Semantic-style commit messages are preferred:

- `fix: handle failed local fallback`
- `feat: add upstream health endpoint`
- `docs: expand configuration guide`
- `test: cover admin token auth`
- `chore: update CI workflow`

Avoid mixing unrelated work in one pull request. A routing bug fix, a UI redesign, and documentation cleanup should usually be separate PRs.

## Testing Expectations

At minimum, run:

```bash
go test ./...
go vet ./...
go build ./cmd/osb
```

Add tests when changing:

- routing policy behavior;
- retry and failover classification;
- config parsing or validation;
- admin API auth or mutation endpoints;
- streaming behavior;
- secret handling;
- CLI commands;
- UI routes or proxy behavior.

For behavior changes, include tests that prove both the success path and the important failure path.

## Documentation Expectations

Update documentation when behavior changes. Good documentation should explain:

- what the feature does;
- when a user should use it;
- the relevant config fields or commands;
- security implications;
- limitations and operational tradeoffs.

Prefer concrete examples over vague descriptions. If a flow has multiple components, use a Mermaid diagram when it helps readers understand the system.

## Design Principles

### Local-first by default

The default experience should favor local Ollama usage. Cloud upstreams are optional extensions for overflow, fallback, or explicit routing.

### Safe network exposure

Localhost binding should remain the default. Any feature that encourages remote exposure must include clear security controls and documentation.

### Predictable failure behavior

Retries and failover should be explicit and explainable. Avoid hidden behavior that makes requests difficult to debug.

### Small, observable runtime state

Runtime state should be visible through CLI or admin APIs where practical. Users should be able to understand which upstream was used and why another upstream was skipped.

### Configuration clarity

Configuration should be stable, validated at load time, and documented with examples.

## Pull Request Checklist

Before submitting a PR, verify:

- The change is focused and described clearly.
- Tests were added or updated where needed.
- `go test ./...` passes.
- `go vet ./...` passes.
- `go build ./cmd/osb` passes.
- Documentation was updated for user-visible behavior.
- No secrets, tokens, private URLs, or local machine paths were committed.
- Security implications were considered and documented.

## Security-Sensitive Changes

Do not open a public PR for a suspected vulnerability before it is reported privately. Follow `SECURITY.md` for private reporting.

Security-sensitive code includes:

- admin authentication;
- secret storage and loading;
- request forwarding headers;
- upstream authorization;
- network listener binding;
- logging and error messages that could expose tokens.

## Code Style

- Follow standard Go formatting with `gofmt`.
- Prefer clear, direct code over clever abstractions.
- Keep comments short and useful.
- Use structured parsers and typed config where possible.
- Avoid broad refactors unless they are necessary for the change.

## Review Process

Review focuses on:

- correctness;
- test coverage;
- security and operational risk;
- compatibility with existing config and CLI behavior;
- clarity of documentation.

Maintainers may ask for smaller PRs when a change is too broad. That is to keep reviews accurate and the project stable.

## Community Standards

All contributors must follow `CODE_OF_CONDUCT.md`. Keep discussion technical, respectful, and focused on making the project better.
