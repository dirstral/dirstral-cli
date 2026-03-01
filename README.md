# dirstral

Terminal-first CLI inspired by opencode, built in Go.

## What it does

- `dirstral` opens a starter screen with mode selection:
  - Breeze (text chat)
  - Tempest (voice loop)
  - Lighthouse (host dir2mcp)
- `dirstral lighthouse` launches and manages a local `dir2mcp` process.
- `dirstral breeze` connects to an MCP server and uses dir2mcp tools.
- `dirstral tempest` records audio, transcribes with ElevenLabs, asks via MCP, and can play TTS.

## Build and run

```bash
make build
make run
```

Install globally so you can run `dirstral` directly:

```bash
make install
```

## Quick start

1. Start dir2mcp host mode:

```bash
dirstral lighthouse up --dir ../dir2mcp
```

2. In another terminal, start text mode:

```bash
dirstral breeze --mcp http://127.0.0.1:8087/mcp
```

3. Or just run:

```bash
dirstral
```

Then select a mode from the startup screen.

## Breeze JSON mode

Use JSON mode when you want machine-friendly output instead of interactive TUI output:

```bash
dirstral breeze --json --mcp http://127.0.0.1:8087/mcp
```

JSON mode emits one NDJSON event per line with a stable envelope:

```json
{"version":"v1","type":"<event_type>","data":{...}}
```

Envelope contract (`v1`):

- `version` (string, required): schema version. Current value is `v1`.
- `type` (string, required): event type.
- `data` (object, required): event payload for the given type.

Known event types:

- `session`: first event for each run.
  - `data.mcp_url` (string)
  - `data.transport` (string)
  - `data.model` (string)
  - `data.session_id` (string)
- `tool_result`: successful tool execution.
  - `data.tool` (string)
  - `data.args` (object)
  - `data.is_error` (boolean)
  - `data.output` (string)
  - `data.citations` (array of strings)
  - `data.structured_content` (object)
- `help`: emitted for `/help`.
  - `data.text` (string)
- `approval_required`: emitted when a tool is not auto-approved.
  - `data.tool` (string)
  - `data.approved` (boolean, currently `false`)
- `error`: parse or execution failure.
  - `data.message` (string)
  - `data.tool` (string, optional)
- `exit`: emitted before loop exits (for `/quit` or `/exit`).
  - `data.reason` (string, currently `user`)

Citation format:

- `tool_result.data.citations` is a list of rendered citation strings (for example `src/main.go:3-9` or page spans).

Compatibility note:

- Consumers should branch on `version`. New event/data fields may be added in a backward-compatible way, but `version: "v1"` envelope keys remain stable.

## Config precedence

`dirstral` loads configuration in this order:

1. Shell environment variables
2. `.env.local`
3. `.env`
4. `~/.config/dirstral/config.toml`

Example `~/.config/dirstral/config.toml`:

```toml
[mcp]
url = "http://127.0.0.1:8087/mcp"
transport = "streamable-http"

model = "mistral-small-latest"
verbose = false

[host]
listen = "127.0.0.1:8087"
mcp_path = "/mcp"

[elevenlabs]
base_url = "https://api.elevenlabs.io"
voice = "Rachel"
```

## Environment variables

- `DIRSTRAL_MCP_URL`
- `DIRSTRAL_MCP_TRANSPORT`
- `DIRSTRAL_MODEL`
- `DIRSTRAL_VERBOSE`
- `DIRSTRAL_HOST_LISTEN`
- `DIRSTRAL_HOST_MCP_PATH`
- `DIRSTRAL_VOICE`
- `ELEVENLABS_API_KEY`
- `ELEVENLABS_BASE_URL`
- `DIR2MCP_AUTH_TOKEN`

## Notes

- `dir2mcp/` is treated as read-only by this project.
- Tempest uses external binaries for local media handling:
  - record: `ffmpeg`
  - playback: `afplay` (macOS) or `ffplay`
