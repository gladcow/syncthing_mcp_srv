package syncthing

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

// StartOptions configures starting Syncthing.
type StartOptions struct {
	HomeDir    string
	ConfigDir  string
	DataDir    string
	GUIAddress string
	ExtraArgs  []string
}

// Process manages a Syncthing OS process lifecycle.
type Process struct {
	cmd   *exec.Cmd
	cmdMu sync.Mutex
}

// Start runs Syncthing in the background with the given options.
func (p *Process) Start(ctx context.Context, opts StartOptions) error {
	p.cmdMu.Lock()
	defer p.cmdMu.Unlock()

	if p.cmd != nil {
		return fmt.Errorf("syncthing already started")
	}

	homeDir := opts.HomeDir
	if homeDir == "" {
		return fmt.Errorf("HomeDir is required to start syncthing")
	}

	args := []string{"-home", homeDir, "-no-browser"}
	if opts.ConfigDir != "" {
		args = append(args, "-config", opts.ConfigDir)
	}
	if opts.DataDir != "" {
		args = append(args, "-data", opts.DataDir)
	}
	if opts.GUIAddress != "" {
		args = append(args, "-gui-address", opts.GUIAddress)
	}
	args = append(args, opts.ExtraArgs...)

	p.cmd = exec.CommandContext(ctx, "syncthing", args...)
	if err := p.cmd.Start(); err != nil {
		p.cmd = nil
		return fmt.Errorf("failed to start syncthing: %w", err)
	}
	return nil
}

// Wait blocks until the Syncthing process exits.
func (p *Process) Wait() error {
	p.cmdMu.Lock()
	cmd := p.cmd
	p.cmdMu.Unlock()

	if cmd == nil {
		return fmt.Errorf("syncthing was not started")
	}
	return cmd.Wait()
}

// Kill terminates the Syncthing process.
func (p *Process) Kill() error {
	p.cmdMu.Lock()
	defer p.cmdMu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	err := p.cmd.Process.Kill()
	p.cmd = nil
	return err
}
