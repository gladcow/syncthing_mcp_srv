package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syncthing_mcp_server/internal/mcp_resources"

	"syncthing_mcp_server/internal/mcp_tools"
	"syncthing_mcp_server/internal/syncthing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	homeDir := os.Getenv("SYNCTHING_HOME")
	if homeDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			homeDir = filepath.Join(home, ".config", "syncthing")
		}
	}

	baseURL := os.Getenv("SYNCTHING_URL")
	if baseURL == "" {
		baseURL = syncthing.DefaultBaseURL
	}

	svc := syncthing.NewService(homeDir, baseURL)

	s := server.NewMCPServer(
		"Sync with SyncThing",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Sync tool
	syncTool := mcp.NewTool("sync",
		mcp.WithDescription("Starts Syncthing if required and verifies the given folder is synced with given device"),
		mcp.WithString("device",
			mcp.Required(),
			mcp.Description("Device name to sync with"),
		),
		mcp.WithString("folder",
			mcp.Required(),
			mcp.Description("Folder name to sync with given device"),
		),
	)
	syncHandler := mcp_tools.NewSyncHandler(svc)
	s.AddTool(syncTool, syncHandler.Sync)

	// List of the configured devices
	devices := mcp.NewResource(
		"syncthing://devices",
		"Devices",
		mcp.WithResourceDescription("The list of devices configured with this SyncThing instance"),
		mcp.WithMIMEType("application/json"),
	)
	devicesHandler := mcp_resources.NewDevicesHandler(svc)
	s.AddResource(devices, devicesHandler.Devices)

	// List of the configured folders
	folders := mcp.NewResource(
		"syncthing://folders",
		"Folders",
		mcp.WithResourceDescription("The list of folders configured with this SyncThing instance"),
		mcp.WithMIMEType("application/json"),
	)
	foldersHandler := mcp_resources.NewFoldersHandler(svc)
	s.AddResource(folders, foldersHandler.Folders)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
