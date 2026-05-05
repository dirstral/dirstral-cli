# dirstral-cli

Terminal orchestrator/client UX for Dirstral MCP servers.

## Scope

`dirstral-cli` is a standalone client/orchestrator. It may:
- start/probe/manage compatible MCP servers
- connect over stdio/HTTP
- provide chat/voice workflows and settings UX

It must not import `dir2mcp` implementation internals.
Composition is over MCP protocol boundaries only.
