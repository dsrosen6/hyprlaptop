package hypr

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
)

const (
	runtimeEnv = "XDG_RUNTIME_DIR"
	sigEnv     = "HYPRLAND_INSTANCE_SIGNATURE"
	sockName   = ".socket2.sock"
)

var (
	ErrMissingEnvs = errors.New("missing hyprland envs")
	monitorEvents  = map[string]struct{}{
		"monitoraddedv1":   {},
		"monitoraddedv2":   {},
		"monitorremovedv1": {},
		"monitorremovedv2": {},
	}
)

type SocketConn struct {
	*net.UnixConn
}

type Event struct {
	Name    string
	Payload string
}

func NewSocketConn() (*SocketConn, error) {
	runtime := os.Getenv(runtimeEnv)
	sig := os.Getenv(sigEnv)
	if runtime == "" || sig == "" {
		return nil, ErrMissingEnvs
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

	return &SocketConn{conn}, nil
}

func (h *SocketConn) ListenForEvents(ctx context.Context, out chan<- Event) error {
	scn := bufio.NewScanner(h)
	for scn.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scn.Text()
		ev, err := parseBaseEvent(line)
		if err != nil {
			slog.Error("parse error", "err", err)
			continue
		}

		if _, ok := monitorEvents[ev.Name]; !ok {
			continue
		}

		out <- Event{
			Name:    ev.Name,
			Payload: ev.Payload,
		}
	}

	if err := scn.Err(); err != nil {
		return fmt.Errorf("error scanning: %w", err)
	}

	return nil
}

func parseBaseEvent(line string) (*Event, error) {
	parts := strings.SplitN(line, ">>", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid event: %q", line)
	}

	return &Event{
		Name:    parts[0],
		Payload: parts[1],
	}, nil
}
