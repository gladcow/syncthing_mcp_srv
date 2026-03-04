package syncthing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"context"
)

const (
	DefaultBaseURL = "http://127.0.0.1:8384"
)

// Folder represents a Syncthing folder configuration.
type Folder struct {
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Path    string         `json:"path"`
	Type    string         `json:"type"`
	Devices []FolderDevice `json:"devices"`
	Paused  bool           `json:"paused"`
}

// FolderDevice is a device reference within a folder.
type FolderDevice struct {
	DeviceID     string `json:"deviceID"`
	IntroducedBy string `json:"introducedBy"`
}

// Device represents a Syncthing device configuration.
type Device struct {
	DeviceID    string   `json:"deviceID"`
	Name        string   `json:"name"`
	Addresses   []string `json:"addresses"`
	Compression string   `json:"compression"`
	Paused      bool     `json:"paused"`
}

// SyncStateResult represents the sync completion state for a folder/device pair.
type SyncStateResult struct {
	Completion  float64 `json:"completion"`
	NeedBytes   int64   `json:"needBytes"`
	NeedItems   int64   `json:"needItems"`
	NeedDeletes int64   `json:"needDeletes"`
	GlobalBytes int64   `json:"globalBytes"`
	GlobalItems int64   `json:"globalItems"`
	RemoteState string  `json:"remoteState"`
	Sequence    int64   `json:"sequence"`
}

// Client is a Syncthing REST API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a client for an existing Syncthing instance.
func New(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Shutdown sends a shutdown request to Syncthing via the REST API.
func (c *Client) Shutdown(ctx context.Context) error {
	return c.doRequest(ctx, http.MethodPost, "/rest/system/shutdown", nil)
}

// Folders returns all configured folders.
func (c *Client) Folders(ctx context.Context) ([]Folder, error) {
	var out []Folder
	if err := c.doRequestJSON(ctx, http.MethodGet, "/rest/config/folders", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Devices returns all configured devices.
func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	var out []Device
	if err := c.doRequestJSON(ctx, http.MethodGet, "/rest/config/devices", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SyncState returns the sync completion state for the given folder and device.
// Use empty folderID for all folders; use empty deviceID for the local device.
func (c *Client) SyncState(ctx context.Context, folderID, deviceID string) (*SyncStateResult, error) {
	path := "/rest/db/completion"
	if folderID != "" || deviceID != "" {
		u, err := url.Parse(path)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		if folderID != "" {
			q.Set("folder", folderID)
		}
		if deviceID != "" {
			q.Set("device", deviceID)
		}
		u.RawQuery = q.Encode()
		path = u.String()
	}

	var out SyncStateResult
	if err := c.doRequestJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Health checks if Syncthing is reachable (no auth required).
func (c *Client) Health(ctx context.Context) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return err
	}
	ref, err := url.Parse("/rest/noauth/health")
	if err != nil {
		return err
	}
	u := base.ResolveReference(ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}
	return nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) error {
	_, err := c.doRequestRaw(ctx, method, path, body)
	return err
}

func (c *Client) doRequestJSON(ctx context.Context, method, path string, body io.Reader, out interface{}) error {
	respBody, err := c.doRequestRaw(ctx, method, path, body)
	if err != nil {
		return err
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) doRequestRaw(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	ref, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	reqURL := base.ResolveReference(ref)

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid API key (401)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed: %s", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return respBody, nil
}

var apiKeyRE = regexp.MustCompile(`<apikey>([^<]+)</apikey>`)

// ReadAPIKeyFromConfig reads the API key from a Syncthing config.xml file.
func ReadAPIKeyFromConfig(homeDir string) (string, error) {
	configPath := filepath.Join(homeDir, "config.xml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}
	m := apiKeyRE.FindSubmatch(content)
	if m == nil {
		return "", fmt.Errorf("apikey not found in config")
	}
	return string(m[1]), nil
}
