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
