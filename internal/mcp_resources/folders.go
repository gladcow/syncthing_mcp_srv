package mcp_resources

import (
	"context"
	"encoding/json"
	"fmt"
	"syncthing_mcp_server/internal/syncthing"

	"github.com/mark3labs/mcp-go/mcp"
)

type FoldersHandler struct {
	svc *syncthing.Service
}

func NewFoldersHandler(svc *syncthing.Service) *FoldersHandler {
	return &FoldersHandler{svc: svc}
}

func (h *FoldersHandler) Folders(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	folders, err := h.svc.Folders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	data, err := json.Marshal(folders)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal folders: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "syncthing://folders",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}
