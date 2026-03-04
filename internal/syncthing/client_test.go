package syncthing

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestReadAPIKeyFromConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.xml")
	content := `<configuration><gui><apikey>test-key-123</apikey></gui></configuration>`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	key, err := ReadAPIKeyFromConfig(dir)
	if err != nil {
		t.Fatalf("ReadAPIKeyFromConfig: %v", err)
	}
	if key != "test-key-123" {
		t.Errorf("got key %q, want test-key-123", key)
	}
}

func TestReadAPIKeyFromConfig_NotFound(t *testing.T) {
	_, err := ReadAPIKeyFromConfig("/nonexistent")
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func requireSyncthing(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("syncthing"); err != nil {
		t.Skip("syncthing not installed, skipping e2e test")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// startSyncthing is a test helper that generates config, starts a Process,
// creates a Client, and waits for readiness.
func startSyncthing(t *testing.T, ctx context.Context) (*Process, *Client) {
	t.Helper()

	homeDir := t.TempDir()
	port := freePort(t)
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)

	gen := exec.CommandContext(ctx, "syncthing", "-generate", homeDir)
	if err := gen.Run(); err != nil {
		t.Fatalf("syncthing -generate failed: %v", err)
	}

	apiKey, err := ReadAPIKeyFromConfig(homeDir)
	if err != nil {
		t.Fatalf("read API key: %v", err)
	}

	proc := &Process{}
	if err := proc.Start(ctx, StartOptions{HomeDir: homeDir, GUIAddress: baseURL}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() {
		_ = proc.Kill()
	})

	client := New(baseURL, apiKey)
	waitReady(t, ctx, client)

	return proc, client
}

func TestProcess_StartAndWaitReady(t *testing.T) {
	requireSyncthing(t)
	ctx := context.Background()

	homeDir := t.TempDir()
	port := freePort(t)
	baseURL := "http://127.0.0.1:" + strconv.Itoa(port)

	gen := exec.CommandContext(ctx, "syncthing", "-generate", homeDir)
	if err := gen.Run(); err != nil {
		t.Fatalf("syncthing -generate failed: %v", err)
	}

	apiKey, err := ReadAPIKeyFromConfig(homeDir)
	if err != nil {
		t.Fatalf("read API key: %v", err)
	}

	proc := &Process{}
	if err := proc.Start(ctx, StartOptions{HomeDir: homeDir, GUIAddress: baseURL}); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() {
		_ = proc.Kill()
	})

	client := New(baseURL, apiKey)
	for i := 0; i < 50; i++ {
		if err := client.Health(ctx); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("Syncthing did not become ready within 5s")
}

func TestClient_Folders(t *testing.T) {
	requireSyncthing(t)
	ctx := context.Background()
	_, client := startSyncthing(t, ctx)

	folders, err := client.Folders(ctx)
	if err != nil {
		t.Fatalf("Folders failed: %v", err)
	}
	if len(folders) == 0 {
		t.Log("no folders (unexpected for -generate default)")
	}
}

func TestClient_Devices(t *testing.T) {
	requireSyncthing(t)
	ctx := context.Background()
	_, client := startSyncthing(t, ctx)

	devices, err := client.Devices(ctx)
	if err != nil {
		t.Fatalf("Devices failed: %v", err)
	}
	if len(devices) < 1 {
		t.Fatal("expected at least one device (local)")
	}
}

func TestClient_SyncState(t *testing.T) {
	requireSyncthing(t)
	ctx := context.Background()
	_, client := startSyncthing(t, ctx)

	state, err := client.SyncState(ctx, "", "")
	if err != nil {
		t.Fatalf("SyncState failed: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil SyncStateResult")
	}
	if state.Completion < 0 || state.Completion > 100 {
		t.Errorf("unexpected completion: %f", state.Completion)
	}
}

func TestClient_Shutdown(t *testing.T) {
	requireSyncthing(t)
	ctx := context.Background()
	proc, client := startSyncthing(t, ctx)

	if err := client.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- proc.Wait()
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Logf("Wait returned: %v (expected after shutdown)", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("process did not exit within 5s after shutdown")
	}
}

func waitReady(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if err := client.Health(ctx); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("Syncthing did not become ready within 5s")
}
