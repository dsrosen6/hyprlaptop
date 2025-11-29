package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
)

const (
	runtimeEnv = "XDG_RUNTIME_DIR"
	sigEnv     = "HYPRLAND_INSTANCE_SIGNATURE"
)

var ErrMissingEnvs = errors.New("missing hyprland envs")

type socketConn struct {
	*net.UnixConn
}

func newConn() (*socketConn, error) {
	runtime := os.Getenv(runtimeEnv)
	sig := os.Getenv(sigEnv)
	if runtime == "" || sig == "" {
		return nil, ErrMissingEnvs
	}

	sock := filepath.Join(runtime, "hypr", sig, ".socket2.sock")
	addr := &net.UnixAddr{
		Name: sock,
		Net:  "unix",
	}

	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("connecting to socket: %w", err)
	}

	return &socketConn{conn}, nil
}

func (c *socketConn) listen() error {
	sc := bufio.NewScanner(c)
	for sc.Scan() {
		line := sc.Text()
		if err := handleLine(line); err != nil {
			fmt.Printf("Error handline line %s: %v\n", line, err)
		}
	}

	if err := sc.Err(); err != nil {
		return fmt.Errorf("error scanning: %w", err)
	}

	return nil
}

func handleLine(line string) error {
	event, err := parseBaseEvent(line)
	if err != nil {
		return fmt.Errorf("parsing event: %w", err)
	}
	slog.Debug("event received", "name", event.name, "payload", event.payload)
	switch event.name {
	case "monitoraddedv2":
		n, err := extractMonitorName(event.payload)
		if err != nil {
			logExtractErr(err)
		}
		slog.Debug("got monitor added event", "monitor_name", n)
	case "monitorremovedv2":
		n, err := extractMonitorName(event.payload)
		if err != nil {
			logExtractErr(err)
		}
		slog.Debug("got monitor removed event", "monitor_name", n)
	}

	return nil
}

func logExtractErr(err error) {
	slog.Error("extracting monitor name from event", "error", err)
}
