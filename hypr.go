package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	binaryName       = "hyprctl"
	unknownReqOutput = "unknown request"
	runtimeEnv       = "XDG_RUNTIME_DIR"
	sigEnv           = "HYPRLAND_INSTANCE_SIGNATURE"
	sockName         = ".socket2.sock"
)

var (
	errUnknownRequest = errors.New(unknownReqOutput)
	errMissingEnvs    = errors.New("missing hyprland envs")
)

type (
	hyprClient struct {
		binaryPath string
	}

	monitor struct {
		monitorIdentifiers
		monitorSettings
	}

	monitorIdentifiers struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
	}

	monitorSettings struct {
		Width       int64   `json:"width,omitempty"`
		Height      int64   `json:"height,omitempty"`
		RefreshRate float64 `json:"refreshRate,omitempty"`
		X           int64   `json:"x,omitempty"`
		Y           int64   `json:"y,omitempty"`
		Scale       float64 `json:"scale,omitempty"`
	}

	monitorUpdateParams struct {
		enableOrUpdate []monitor
		disable        []monitor
		noChanges      []monitor
	}

	hyprSocketConn struct {
		*net.UnixConn
	}
)

func newHyprctlClient() (*hyprClient, error) {
	bp, err := exec.LookPath(binaryName)
	if err != nil {
		return nil, fmt.Errorf("finding full hyprctl binary path: %w", err)
	}

	return &hyprClient{binaryPath: bp}, nil
}

func newHyprSocketConn() (*hyprSocketConn, error) {
	runtime := os.Getenv(runtimeEnv)
	sig := os.Getenv(sigEnv)
	if runtime == "" || sig == "" {
		return nil, errMissingEnvs
	}

	sock := filepath.Join(runtime, "hypr", sig, sockName)
	addr := &net.UnixAddr{
		Name: sock,
		Net:  "unix",
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to socket: %w", err)
	}

	return &hyprSocketConn{conn}, nil
}

func newMonitorUpdateParams(enable, disable, noChanges []monitor) *monitorUpdateParams {
	if enable == nil {
		enable = []monitor{}
	}

	if disable == nil {
		disable = []monitor{}
	}

	return &monitorUpdateParams{
		enableOrUpdate: enable,
		disable:        disable,
		noChanges:      noChanges,
	}
}

func (h *hyprClient) runCommandWithUnmarshal(args []string, v any) error {
	a := append([]string{"-j"}, args...)
	out, err := h.runCommand(a)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("unmarshaling json: %w", err)
	}

	return nil
}

func (h *hyprClient) runCommand(args []string) ([]byte, error) {
	cmd := exec.Command(h.binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running command: %w", err)
	}

	out := stdout.Bytes()
	errStr := strings.TrimSpace(stderr.String())
	if errStr != "" {
		return nil, errors.New(errStr)
	}

	return out, checkForErr(string(out))
}

func (h *hyprClient) listMonitors() ([]monitor, error) {
	var monitors []monitor
	if err := h.runCommandWithUnmarshal([]string{"monitors"}, &monitors); err != nil {
		return nil, err
	}

	return monitors, nil
}

func (h *hyprClient) bulkUpdateMonitors(p *monitorUpdateParams) error {
	if p == nil {
		return errors.New("receieved nil params")
	}

	tm := len(p.enableOrUpdate) + len(p.disable)
	errch := make(chan error, tm)

	var wg sync.WaitGroup
	for _, m := range p.enableOrUpdate {
		wg.Go(func() {
			if err := h.enableOrUpdateMonitor(m); err != nil {
				errch <- fmt.Errorf("enabling monitor %s: %w", m.Name, err)
			}
			slog.Debug("bulkUpdateMonitors: display enabled or updated", "name", m.Name)
		})
	}

	for _, m := range p.disable {
		wg.Go(func() {
			if err := h.disableMonitor(m); err != nil {
				errch <- fmt.Errorf("disabling monitor %s: %w", m.Name, err)
			}
			slog.Debug("bulkUpdateMonitors: display disabled", "name", m.Name)
		})
	}

	go func() {
		wg.Wait()
		close(errch)
	}()

	hasErr := false
	for err := range errch {
		hasErr = true
		slog.Error("bulkUpdateMonitors", "error", err)
	}

	if hasErr {
		return errors.New("one or more update failed; see logs")
	}

	return nil
}

func (h *hyprClient) enableOrUpdateMonitor(m monitor) error {
	args := []string{"keyword", "monitor", monitorToConfigString(m)}
	if _, err := h.runCommand(args); err != nil {
		return err
	}

	return nil
}

func (h *hyprClient) disableMonitor(m monitor) error {
	args := []string{"keyword", "monitor", m.Name + ",", "disable"}
	if _, err := h.runCommand(args); err != nil {
		return err
	}

	return nil
}

func monitorToConfigString(m monitor) string {
	res := fmt.Sprintf("%dx%d", m.Width, m.Height)
	res = fmt.Sprintf("%s@%f", res, m.RefreshRate)
	xy := fmt.Sprintf("%dx%d", m.X, m.Y)
	scale := fmt.Sprintf("%f", m.Scale)
	return fmt.Sprintf("%s,%s,%s,%s", m.Name, res, xy, scale)
}

func checkForErr(out string) error {
	out = strings.TrimSpace(out)
	switch out {
	case unknownReqOutput:
		return errUnknownRequest
	default:
		return nil
	}
}
