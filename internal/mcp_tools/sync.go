package mcp_tools

import (
	"context"
	"fmt"
	"syncthing_mcp_server/internal/syncthing"

	"github.com/mark3labs/mcp-go/mcp"
)

type SyncHandler struct {
	svc *syncthing.Service
}

func NewSyncHandler(svc *syncthing.Service) *SyncHandler {
	return &SyncHandler{svc: svc}
}

func (h *SyncHandler) Sync(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deviceName, err := request.RequireString("device")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	folderName, err := request.RequireString("folder")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := h.svc.EnsureRunning(ctx); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot connect to syncthing: %s", err)), nil
	}

	device, err := h.svc.DeviceByName(ctx, deviceName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	folder, err := h.svc.FolderByName(ctx, folderName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	deviceInFolder := false
	for _, fd := range folder.Devices {
		if fd.DeviceID == device.DeviceID {
			deviceInFolder = true
			break
		}
	}
	if !deviceInFolder {
		return mcp.NewToolResultError(fmt.Sprintf(
			"device %q (%s) is not shared with folder %q (%s)",
			device.Name, device.DeviceID, folder.Label, folder.ID,
		)), nil
	}

	state, err := h.svc.WaitForSync(ctx, folder.ID, device.DeviceID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"sync failed for folder %q with device %q: %s",
			folder.Label, device.Name, err,
		)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"folder %q is fully synced with device %q (completion: %.1f%%)",
		folder.Label, device.Name, state.Completion,
	)), nil
}
