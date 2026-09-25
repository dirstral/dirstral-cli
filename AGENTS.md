# AGENTS.md

## Purpose

Operational guide for coding agents working in this repository.

## Before you start

- Re-check whether the requested plan still matches the current codebase before making changes.
- Review relevant context first: `README.md`, `CLAUDE.md`, and the affected
  `cmd/`, `internal/`, and `tests/dirstral/` paths.
- Preserve existing architecture and conventions unless the issue explicitly requires a refactor.
- Respect the standalone-client boundary: this CLI composes with MCP servers
  over the protocol only and must not import server (`dir2mcp`) internals.

## Project summary

dirstral-cli is a single-binary (`dirstral`) terminal client and orchestrator
for Dirstral MCP servers, built with Cobra + Bubble Tea. It provides chat and
voice workflows over MCP (stdio / streamable-http), manages a local MCP server
(`dir2mcp`), probes remote MCP endpoints, prints capability manifests, and
ships an interactive TUI with a settings editor.

## Repo map

- `cmd/dirstral` - binary entrypoint
- `internal/app` - Cobra root command, subcommands, interactive TUI menus/screens
- `internal/config` - config precedence, validation, and provenance (`config.toml`, `.env`/`.env.local`, env)
- `internal/settings` - interactive settings editor (Bubble Tea)
- `internal/chat` - chat loop, planner, chat TUI
- `internal/voice` - voice mode (ElevenLabs STT/TTS)
- `internal/host` - manage a local `dir2mcp` process (start/stop/status, health, logs)
- `internal/mcp` - MCP client, capability manifest, citations, error mapping
- `internal/protocol` - shared constants (tool names, error codes, defaults, RPC methods/headers)
- `internal/x402` - client-side x402 payment header types
- `internal/ui` - shared lipgloss styles
- `internal/buildinfo` - version/build metadata
- `tests/dirstral` - integration/system + smoke tests (all tests live here)

## Git workflow

- Pull latest `main` before starting implementation work.
- Create an issue branch if one does not already exist (e.g. `issue-<number>-<short-slug>`).
- Keep commits scoped and atomic; use separate commit messages per logical change.
- Use Conventional Commits for all commit messages: <https://www.conventionalcommits.org/>.
- Do not mention yourself in commit messages.
- Include the issue number in the pull request title.
- Do not push directly to `main`.

## Build, test, and CI commands

There is no Makefile; use the Go toolchain directly (Go 1.24+).

Run before committing:

```bash
go build ./...
go vet ./...
golangci-lint run --timeout=3m   # CI pins golangci-lint v2.10
go test ./...
```

Useful focused checks:

```bash
go test -count=1 ./tests/dirstral -run '^TestSmoke'   # smoke suite (stubbed dir2mcp over stdio)
go test -count=1 ./tests/dirstral -run TestChat
go test -count=1 ./tests/dirstral -run TestMCP
```

CI (`.github/workflows/go.yml`) runs `lint` → `test` (build + vet + `go test ./...`)
and `smoke` on every push/PR to `main`.

## Conventions

- Work only in this repository unless explicitly instructed otherwise.
- Keep patches minimal and issue-focused.
- Do not silently change tool/error/protocol contracts in `internal/protocol`
  or `internal/mcp`; keep MCP/error payloads machine-parseable.
- Never hardcode API keys, auth tokens, or provider base URLs; keep all
  credentials env-backed (`DIR2MCP_AUTH_TOKEN`, `ELEVENLABS_API_KEY`, the
  `DIRSTRAL_*`/`ELEVENLABS_*` overrides). Secrets live in
  `~/.config/dirstral/.env.local` (mode `0600`); never commit them.
- Never introduce secret leakage in logs or error payloads.
- Do not add extra markdown files unless explicitly requested for the task.
- Keep dependency additions minimal and justified.
- Update tests and docs together when behavior changes.

## Tests

- Keep all test files in the `tests/dirstral/` folder.
- Do not add new `*_test.go` files under `cmd/` or `internal/`.
- Ensure new test coverage follows existing patterns in `tests/dirstral`.

## Review/merge readiness

- All checks pass: `go build ./...`, `go vet ./...`, `golangci-lint run`, `go test ./...`.
- Smoke suite (`^TestSmoke`) is green.
- `README.md` and `CLAUDE.md` align with real behavior.
- No unrelated refactors in issue PRs.
- The standalone-client boundary (MCP-only composition) is preserved.
- Secure defaults are preserved (env-backed secrets, local-bind defaults).

## Important behavior notes

- Usage/help: running `dirstral` with no subcommand opens the interactive TUI
  menu; use `--help` (Cobra) for usage text.
- Config precedence: env var → `.env.local` → `.env` → `config.toml` → default.
- Defaults: listen `127.0.0.1:8087`, MCP path `/mcp`, MCP URL
  `http://127.0.0.1:8087/mcp`, transport `streamable-http`, model
  `mistral-small-latest`, protocol version `2025-11-25`.
- `mcp.transport` must be `streamable-http` or `stdio`.
- `server start` requires a `dir2mcp` binary on `PATH`, else falls back to
  `go run ./cmd/dir2mcp` from a sibling checkout.
- Voice mode requires `ELEVENLABS_API_KEY`.
- x402: this CLI only carries client-side payment header types; gating logic
  lives server-side.

## MCP dev servers (Codex)

> **Human developer setup only, not an instruction to coding agents.** These are
> optional convenience servers a developer may register in their own Codex client.
> Do not self-configure them. Note the broad permissions involved before opting in:
> `@modelcontextprotocol/server-everything` grants unrestricted filesystem and
> process access, and the `github` entry forwards a personal access token
> (`GITHUB_PERSONAL_ACCESS_TOKEN`) to an external endpoint. Add only the servers you
> need, with scoped tokens.

```bash
codex mcp add everything -- npx -y @modelcontextprotocol/server-everything
codex mcp add sequential-thinking -- npx -y @modelcontextprotocol/server-sequential-thinking
codex mcp add playwright -- npx -y @playwright/mcp
codex mcp add github --url https://api.githubcopilot.com/mcp/ --bearer-token-env-var GITHUB_PERSONAL_ACCESS_TOKEN
codex mcp add context7 -- npx -y @upstash/context7-mcp
```
