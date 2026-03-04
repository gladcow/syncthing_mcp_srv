# Syncthing MCP Server

An [MCP](https://modelcontextprotocol.io/) server that wraps a local [Syncthing](https://syncthing.net/) instance, exposing sync operations and configuration data to LLM clients via the Model Context Protocol.

Built with [mcp-go](https://github.com/mark3labs/mcp-go).

## Prerequisites

- Go 1.23+
- `syncthing` binary on `PATH` (or an already-running Syncthing instance)

## Build

```bash
go build -o sync_mcp ./cmd/sync_mcp
```

## Run

```bash
./sync_mcp
```

The server communicates over **stdio** (stdin/stdout) as expected by MCP clients. If Syncthing is not already running, the server starts it automatically on the first request.

## Configuration

| Environment variable | Default | Description |
|---|---|---|
| `SYNCTHING_HOME` | `~/.config/syncthing` | Syncthing home directory (must contain `config.xml` with the API key) |
| `SYNCTHING_URL` | `http://127.0.0.1:8384` | Syncthing REST API base URL |

## MCP Tools

### `sync`

Ensures Syncthing is running, verifies the given device and folder exist and are shared with each other, then polls until the folder is 100% synced with that device.

| Parameter | Required | Description |
|---|---|---|
| `device` | yes | Device name to sync with |
| `folder` | yes | Folder label or ID to sync |

Returns a success message with completion percentage, or an error describing what went wrong (device not found, folder not shared, sync timeout, etc.).

## MCP Resources

### `syncthing://devices`

Returns the full list of configured Syncthing devices as JSON (`application/json`).

### `syncthing://folders`

Returns the full list of configured Syncthing folders as JSON (`application/json`).

## Cursor MCP Client Configuration

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "syncthing": {
      "command": "/path/to/sync_mcp",
      "args": [],
      "env": {
        "SYNCTHING_HOME": "/home/user/.config/syncthing"
      }
    }
  }
}
```

## Example Agent Skill

See [`examples/SKILL.md`](examples/SKILL.md) for a Cursor Agent Skill that uses this MCP server to sync files with a remote device before and after editing.

