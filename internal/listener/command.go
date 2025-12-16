package listener

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var CommandSockName = "hyprlaptop.sock"

// commandListener listens for CLI-specific commands besides "listen" and performs
// actions when required.
func (l *Listener) commandListener(ctx context.Context, events chan<- Event) error {
	sock := filepath.Join(os.TempDir(), CommandSockName)

	// remove exiting file if it already exists
	_ = os.Remove(sock)

	ln, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("command listener: listen unix socket: %w", err)
	}

	defer func() {
		if err := ln.Close(); err != nil {
			slog.Error("command listener: closing hyprlaptop socket", "error", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			conn, err := ln.Accept()
			if err != nil {
				continue
			}

			go func() {
				defer func() {
					if err := conn.Close(); err != nil {
						slog.Error("command listener: closing socket conn", "error", err)
					} else {
						slog.Debug("command listener: socket conn closed")
					}
				}()

				buf, _ := io.ReadAll(conn)
				msg := strings.TrimSpace(string(buf))

				switch msg {
				case string(LidSwitchEvent):
					events <- Event{Type: LidSwitchEvent}
				case string(IdleWakeEvent):
					events <- Event{Type: IdleWakeEvent}
				case string(DisplayUnknownEvent):
					events <- Event{Type: DisplayUnknownEvent}
				default:
					slog.Warn("command listener: got unknown message", "msg", msg)
				}
			}()
		}
	}
}
