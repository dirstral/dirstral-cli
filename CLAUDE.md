# CLAUDE.md

## Project

dirstral-cli is a terminal-first Go client and orchestrator for Dirstral MCP
servers. It ships a single `dirstral` binary that drives chat and voice
workflows over MCP, manages a local MCP server (today: `dir2mcp`), probes
remote MCP endpoints, prints capability manifests, and offers an interactive
TUI (Cobra + Bubble Tea) plus a settings editor.

It is a standalone client/orchestrator: it composes with servers **only over
the MCP protocol boundary** (stdio or streamable-http). It must not import
`dir2mcp` (or any server) implementation internals: see `README.md`.

## Repository layout

- `cmd/dirstral`: binary entrypoint (`main.go`)
- `internal/app`: Cobra root command, subcommands, and the interactive TUI menus/screens
- `internal/config`: config load/merge/validation (`config.toml`, `.env`/`.env.local`, env vars) with precedence + provenance
- `internal/settings`: interactive Bubble Tea settings editor
- `internal/chat`: chat loop, planner, and chat TUI
- `internal/voice`: voice mode (STT/TTS via ElevenLabs)
- `internal/host`: manage a local `dir2mcp` process (start/stop/status, health, log streaming, `connection.json` discovery)
- `internal/mcp`: MCP client, capability manifest, citations, error mapping
- `internal/protocol`: shared constants (tool names, error codes, defaults, RPC method/header names)
- `internal/x402`: client-side x402 payment header types (PAYMENT-REQUIRED/SIGNATURE/RESPONSE)
- `internal/ui`: shared lipgloss styles
- `internal/buildinfo`: version/build metadata
- `tests/dirstral`: integration/system + smoke test suite (all tests live here)

## Build and test

There is **no Makefile**; use the Go toolchain directly (Go 1.24+).

- Build: `go build ./...`
- Vet: `go vet ./...`
- Test: `go test ./...`
- Lint: `golangci-lint run --timeout=3m` (CI pins golangci-lint v2.10)
- Smoke only: `go test -count=1 ./tests/dirstral -run '^TestSmoke'`

CI (`.github/workflows/go.yml`) runs three jobs on push/PR to `main`:
`lint` (golangci-lint) → then `test` (build + vet + `go test ./...`) and
`smoke` (`^TestSmoke` in `tests/dirstral`).

The smoke suite stands up the client against a fake/stubbed `dir2mcp` over
stdio, so it needs no live server or provider credentials.

## Configuration

Config is resolved with precedence: env var → `.env.local` → `.env` →
`config.toml` → built-in default.

- `config.toml`: `~/.config/dirstral/config.toml` (`os.UserConfigDir()`)
- Secrets: `~/.config/dirstral/.env.local` (written `0600`; never commit)
- Keys: `mcp.url`, `mcp.transport` (`streamable-http`|`stdio`), `model`,
  `verbose`, `host.listen`, `host.mcp_path`, `elevenlabs.base_url`,
  `elevenlabs.voice`
- Secret env vars: `DIR2MCP_AUTH_TOKEN`, `ELEVENLABS_API_KEY`
- Override env vars: `DIRSTRAL_MCP_URL`, `DIRSTRAL_MCP_TRANSPORT`,
  `DIRSTRAL_MODEL`, `DIRSTRAL_VERBOSE`, `DIRSTRAL_HOST_LISTEN`,
  `DIRSTRAL_HOST_MCP_PATH`, `DIRSTRAL_VOICE`, `ELEVENLABS_BASE_URL`
- Defaults: listen `127.0.0.1:8087`, MCP path `/mcp`, MCP URL
  `http://127.0.0.1:8087/mcp`, transport `streamable-http`, model
  `mistral-small-latest`, MCP protocol version `2025-11-25`

## Subcommands

- `dirstral` (no args): interactive TUI menu (Chat, Voice, MCP Server, Settings, Exit)
- `dirstral chat`: chat mode (`--mcp`, `--transport`, `--model`, `--verbose`, `--json`)
- `dirstral voice`: voice mode (`--mcp`, `--voice`, `--device`, `--mute`, `--elevenlabs-base-url`, `--verbose`)
- `dirstral server start|status|stop|remote`: manage local `dir2mcp` host / probe remote MCP
- `dirstral manifest`: print the MCP capability manifest (`--mcp`, `--transport`, `--json`, `--verbose`)

## Releasing

Release artifacts are produced by GoReleaser (`.goreleaser.yml`): project
`dirstral-cli`, binary `dirstral` from `./cmd/dirstral`, `tar.gz` archives,
`prerelease: auto`. Tag a commit on `main` to cut a release.

## Working conventions

- Keep changes scoped to the issue.
- Preserve the standalone-client boundary: talk to servers over MCP only; do
  not import server internals.
- Preserve existing tool/error/protocol contracts and structured fields
  (`internal/protocol`, `internal/mcp`).
- Never hardcode credentials, auth tokens, or provider base URLs; keep them env-backed.
- Do not log secrets or raw sensitive payloads.
- Prefer deterministic behavior and explicit error handling.
- If behavior changes, update tests and docs in the same PR.

## PR checklist

- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` pass locally
- [ ] `golangci-lint run` is clean
- [ ] Smoke suite (`go test -count=1 ./tests/dirstral -run '^TestSmoke'`) passes
- [ ] New/changed behavior has test coverage in `tests/dirstral`
- [ ] `README.md` stays truthful
- [ ] No unrelated files changed

## Known gotchas

- No Makefile: use raw `go` commands (and `golangci-lint` for lint).
- Running `dirstral` with no subcommand launches the interactive TUI menu, not
  usage text; use `dirstral --help` / `dirstral <cmd> --help` for help (Cobra).
- `server start` needs a `dir2mcp` binary on `PATH`; otherwise it falls back to
  `go run ./cmd/dir2mcp` from a sibling checkout, and fails if neither exists.
- `mcp.transport` must be exactly `streamable-http` or `stdio`.
- Voice mode requires `ELEVENLABS_API_KEY` (base URL defaults to `https://api.elevenlabs.io`).
- URL resolution prefers a healthy managed local host when the configured MCP
  URL is empty or loopback; an explicit `--mcp` override always wins.
- All tests live under `tests/dirstral`; do not add `*_test.go` under `cmd/` or `internal/`.
