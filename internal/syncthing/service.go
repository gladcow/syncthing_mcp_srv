package syncthing

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultSyncTimeout  = 10 * time.Minute
	healthPollInterval  = 200 * time.Millisecond
	healthTimeout       = 30 * time.Second
)

// Service manages a Syncthing instance lifecycle and provides
// high-level operations on top of the REST API client.
// It is safe for concurrent use and intended to be shared across
// multiple MCP tool and resource handlers.
type Service struct {
	homeDir string
	baseURL string
	proc    *Process
	client  *Client
	mu      sync.Mutex
}

func NewService(homeDir, baseURL string) *Service {
	return &Service{
		homeDir: homeDir,
		baseURL: baseURL,
	}
}

// EnsureRunning guarantees that Syncthing is reachable via its REST API.
// If it is not already running, the process is started and polled until healthy.
func (s *Service) EnsureRunning(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		if err := s.client.Health(ctx); err == nil {
			return nil
		}
	}

	probe := New(s.baseURL, "")
	if err := probe.Health(ctx); err == nil {
		apiKey, err := ReadAPIKeyFromConfig(s.homeDir)
		if err != nil {
			return fmt.Errorf("syncthing is running but cannot read API key: %w", err)
		}
		s.client = New(s.baseURL, apiKey)
		return nil
	}

	if s.proc == nil {
		s.proc = &Process{}
	}
	if err := s.proc.Start(ctx, StartOptions{HomeDir: s.homeDir}); err != nil {
		return fmt.Errorf("failed to start syncthing: %w", err)
	}

	deadline := time.Now().Add(healthTimeout)
	for time.Now().Before(deadline) {
		if err := probe.Health(ctx); err == nil {
			apiKey, err := ReadAPIKeyFromConfig(s.homeDir)
			if err != nil {
				return fmt.Errorf("syncthing started but cannot read API key: %w", err)
			}
			s.client = New(s.baseURL, apiKey)
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(healthPollInterval):
		}
	}

	return fmt.Errorf("syncthing did not become ready within %s", healthTimeout)
}

// Devices returns all configured devices. EnsureRunning is called first.
func (s *Service) Devices(ctx context.Context) ([]Device, error) {
	if err := s.EnsureRunning(ctx); err != nil {
		return nil, err
	}
	return s.client.Devices(ctx)
}

// Folders returns all configured folders. EnsureRunning is called first.
func (s *Service) Folders(ctx context.Context) ([]Folder, error) {
	if err := s.EnsureRunning(ctx); err != nil {
		return nil, err
	}
	return s.client.Folders(ctx)
}

// DeviceByName finds a device by its display name (case-insensitive).
// Returns a descriptive error listing available names when not found.
func (s *Service) DeviceByName(ctx context.Context, name string) (*Device, error) {
	devices, err := s.Devices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	for i := range devices {
		if strings.EqualFold(devices[i].Name, name) {
			return &devices[i], nil
		}
	}

	names := make([]string, len(devices))
	for i, d := range devices {
		names[i] = d.Name
	}
	return nil, fmt.Errorf("device %q not found; available devices: %s", name, strings.Join(names, ", "))
}

// FolderByName finds a folder by its label or ID (case-insensitive).
// Returns a descriptive error listing available folders when not found.
func (s *Service) FolderByName(ctx context.Context, name string) (*Folder, error) {
	folders, err := s.Folders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	for i := range folders {
		if strings.EqualFold(folders[i].Label, name) || strings.EqualFold(folders[i].ID, name) {
			return &folders[i], nil
		}
	}

	labels := make([]string, len(folders))
	for i, f := range folders {
		if f.Label != "" {
			labels[i] = f.Label
		} else {
			labels[i] = f.ID
		}
	}
	return nil, fmt.Errorf("folder %q not found; available folders: %s", name, strings.Join(labels, ", "))
}

// WaitForSync polls the sync completion state for the given folder and device
// until Completion >= 100 or the timeout expires. The default timeout is 10
// minutes but a shorter deadline set on ctx takes precedence.
func (s *Service) WaitForSync(ctx context.Context, folderID, deviceID string) (*SyncStateResult, error) {
	if err := s.EnsureRunning(ctx); err != nil {
		return nil, err
	}

	timeout := defaultSyncTimeout
	if dl, ok := ctx.Deadline(); ok {
		if remaining := time.Until(dl); remaining < timeout {
			timeout = remaining
		}
	}
	deadline := time.Now().Add(timeout)

	for {
		state, err := s.client.SyncState(ctx, folderID, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get sync state: %w", err)
		}

		if state.Completion >= 100 {
			return state, nil
		}

		if time.Now().After(deadline) {
			return state, fmt.Errorf(
				"sync timed out after %s; completion: %.1f%% (need %d items, %d bytes)",
				timeout, state.Completion, state.NeedItems, state.NeedBytes,
			)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(defaultPollInterval):
		}
	}
}
