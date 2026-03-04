---
name: syncthing-sync
description: Use when working on files that are shared with a remote device via Syncthing. Syncs the folder before and after making changes to keep both sides up to date.
---

# Syncthing Sync Skill

## When to Use

Use this skill whenever you are about to edit files that live in a Syncthing-shared folder and need to ensure they are synchronized with a specific remote device before and after your changes.

## Prerequisites

The `syncthing` MCP server must be configured in `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "syncthing": {
      "command": "/path/to/sync_mcp",
      "args": []
    }
  }
}
```

## Steps

### 1. Discover devices and folders

Read the MCP resources to find out what is available:

- Read `syncthing://devices` to get the list of configured devices and their names.
- Read `syncthing://folders` to get the list of configured folders, their labels, and paths.

Identify the **device name** and **folder label** that match the user's request. If the user did not specify them, ask which device and folder to sync.

### 2. Sync before editing

Call the `sync` tool to ensure the local copy is fully up to date before making any changes:

```
sync(device: "<device name>", folder: "<folder label>")
```

Wait for the tool to return success. If it returns an error (device not found, folder not shared, timeout), report the error to the user and stop.

### 3. Make changes

Proceed with the requested file edits in the synced folder.

### 4. Sync after editing

Call the `sync` tool again after all changes are complete to push them to the remote device:

```
sync(device: "<device name>", folder: "<folder label>")
```

Report the sync result to the user.
