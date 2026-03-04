package mcp_resources

import (
	"context"
	"encoding/json"
	"fmt"
	"syncthing_mcp_server/internal/syncthing"

	"github.com/mark3labs/mcp-go/mcp"
)

type DevicesHandler struct {
	svc *syncthing.Service
}

func NewDevicesHandler(svc *syncthing.Service) *DevicesHandler {
	return &DevicesHandler{svc: svc}
}

func (h *DevicesHandler) Devices(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	devices, err := h.svc.Devices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	data, err := json.Marshal(devices)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal devices: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "syncthing://devices",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
