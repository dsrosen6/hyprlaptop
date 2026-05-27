package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/dsrosen6/hyprdocked/internal/hypr"
	"github.com/dsrosen6/hyprdocked/internal/power"
	"github.com/godbus/dbus/v5"
)

type (
	listener struct {
		hctlSocketConn *hypr.SocketConn
		lidHandler     *power.LidHandler
		configCh       chan Config
	}

	listenerEvent struct {
		Type    eventType
		Details string
		Done    chan error
	}

	listenerParams struct {
		hyprSockConn *hypr.SocketConn
		lidHandler   *power.LidHandler
		dbusConn     *dbus.Conn
	}

	eventType string
)

var displayEvents = map[string]eventType{
	"monitoradded":   displayAddEvent,
	"monitorremoved": displayRemoveEvent,
}

const (
	displayAddEvent     eventType = "DISPLAY_ADDED"
	displayRemoveEvent  eventType = "DISPLAY_REMOVED"
	displayUnknownEvent eventType = "DISLAY_UNKNOWN_EVENT"
	lidSwitchEvent      eventType = "LID_SWITCH"
	idleCmdEvent        eventType = "IDLE_CMD"
	resumeCmdEvent      eventType = "RESUME_CMD"
	pingCmdEvent        eventType = "PING_CMD"

	cmdSockName         = "hyprdocked.sock"
	defaultSettleWindow = 1
)

func newListener(p listenerParams) (*listener, error) {
	return &listener{
		hctlSocketConn: p.hyprSockConn,
		lidHandler:     p.lidHandler,
		configCh:       make(chan Config, 1),
	}, nil
}

// listenAndHandle starts the hyprdocked listener, which handles hyprctl display add/remove events
// and events from the hyprdocked CLI.
func (a *App) listenAndHandle(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan listenerEvent, 16)
	errc := make(chan error, 1)

	go func() {
		slog.Info("listening for updates")
		if err := a.listener.listen(ctx, events); err != nil {
			errc <- err
			cancel()
		}
	}()

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil // normal shutdown
			}

			// Collect done channels to signal once processing completes.
			var doneChans []chan error
			if ev.Done != nil {
				doneChans = append(doneChans, ev.Done)
			}

			if a.mode == modeIdle && ev.Type != resumeCmdEvent {
				slog.Debug("received event from listener; in idle mode, skipping processing", "type", ev.Type, "details", ev.Details)
				for _, done := range doneChans {
					done <- nil
				}
				continue
			}

			slog.Debug("received event from listener", "type", ev.Type, "details", ev.Details)
			switch ev.Type {
			case resumeCmdEvent:
				slog.Info("resume command received", "source", ev.Details)
				a.mode = modeNormal
			case idleCmdEvent:
				slog.Info("idle command received", "source", ev.Details)
				a.mode = modeIdle
			case pingCmdEvent:
				slog.Info("ping command received")
				for _, done := range doneChans {
					done <- nil
				}
				continue
			}

			// Wait briefly to let the system settle and coalesce any concurrently buffered
			// events (e.g. rapid display add/remove during dock/undock).
			sw := a.Config.SettleWindow
			if sw <= 0 {
				sw = defaultSettleWindow
			}
			settle := time.NewTimer(time.Duration(sw) * time.Second)
		drain:
			for {
				select {
				case <-settle.C:
					break drain
				case extra, ok := <-events:
					if !ok {
						settle.Stop()
						for _, done := range doneChans {
							done <- nil
						}
						return nil
					}
					if extra.Done != nil {
						doneChans = append(doneChans, extra.Done)
					}
					slog.Debug("coalescing event during settle", "type", extra.Type, "details", extra.Details)
					switch extra.Type {
					case resumeCmdEvent:
						a.mode = modeNormal
					case idleCmdEvent:
						a.mode = modeIdle
					}
				}
			}
			settle.Stop()

			// Re-fetch all state from authoritative sources before deciding what to do.
			a.refreshState(ctx)

			var (
				runErr  error
				changed bool
			)
			if !a.ready() {
				slog.Debug("not ready; awaiting initial values")
			} else if a.updating {
				slog.Debug("skipping: mid update")
			} else {
				changed, runErr = a.runUpdater()
				if runErr != nil {
					slog.Error("running updater", "error", runErr)
				}
				a.runPostHooks(changed)
			}

			for _, done := range doneChans {
				done <- runErr
			}

		case cfg := <-a.listener.configCh:
			a.Config = cfg

		case err := <-errc:
			return fmt.Errorf("listener failed: %w", err)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (a *App) refreshState(ctx context.Context) {
	if ds, err := a.hctl.ListMonitors(); err == nil {
		if !reflect.DeepEqual(a.allDisplays, ds) {
			a.allDisplays = ds
			slog.Debug("displays state refreshed", "displays", ds)
		}
	} else {
		slog.Error("refreshing displays", "error", err)
	}

	if ls, err := a.listener.lidHandler.GetCurrentState(ctx); err == nil {
		if a.lidState != ls {
			a.lidState = ls
			slog.Debug("lid state refreshed", "state", ls)
		}
	} else {
		slog.Error("refreshing lid state", "error", err)
	}
}

func (l *listener) listen(ctx context.Context, events chan<- listenerEvent) error {
	errc := make(chan error, 1)
	go func() {
		slog.Debug("listening for hyprland events")
		if err := l.listenHyprctl(ctx, events); err != nil {
			errc <- fmt.Errorf("hyprland listener: %w", err)
		}
	}()

	go func() {
		slog.Debug("listening for lid events")
		if err := l.listenLidEvents(ctx, events); err != nil {
			errc <- fmt.Errorf("lid listener: %w", err)
		}
	}()

	go func() {
		slog.Debug("listening for command events")
		if err := l.listenCommandEvents(ctx, events); err != nil {
			errc <- fmt.Errorf("command listener: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

// listenHyprctl listens for hyprctl events and sends an event if it is a display add or removal.
func (l *listener) listenHyprctl(ctx context.Context, events chan<- listenerEvent) error {
	var lastEvent listenerEvent
	scn := bufio.NewScanner(l.hctlSocketConn)
	for scn.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scn.Text()

			ev, err := parseDisplayEvent(line)
			if err != nil {
				slog.Error("parse error", "err", err)
				continue
			}

			if ev.Type == displayUnknownEvent {
				continue
			}

			// store and check for last event so it doesn't attempt to send an unnecessary event if received
			if reflect.DeepEqual(lastEvent, ev) {
				slog.Debug("hyprctl listener: new event matches last event, no action needed")
				continue
			}

			lastEvent = ev
			events <- ev
		}
	}

	if err := scn.Err(); err != nil {
		return fmt.Errorf("error scanning: %w", err)
	}

	if ctx.Err() == nil {
		return errors.New("hyprland socket closed unexpectedly")
	}
	return nil
}

func (l *listener) listenLidEvents(ctx context.Context, events chan<- listenerEvent) error {
	go func() {
		if err := l.lidHandler.ListenForChanges(ctx); err != nil && err != context.Canceled {
			slog.Error("lid listener stopped", "error", err)
		}
	}()

	for range l.lidHandler.Events {
		select {
		case events <- listenerEvent{Type: lidSwitchEvent}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (l *listener) listenCommandEvents(ctx context.Context, events chan<- listenerEvent) error {
	sock := filepath.Join(os.TempDir(), cmdSockName)

	// remove existing file if it already exists
	_ = os.Remove(sock)

	ln, err := net.Listen("unix", sock)
	if err != nil {
		return fmt.Errorf("command listener: listening to unix socket: %w", err)
	}

	defer func() {
		if err := ln.Close(); err != nil {
			slog.Error("command listener: closing hyprdocked socket", "error", err)
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
					}
				}()

				buf, _ := io.ReadAll(conn)
				msg := strings.TrimSpace(string(buf))

				parts := strings.SplitN(msg, " ", 2)
				cmd := parts[0]
				source := ""
				if len(parts) > 1 {
					source = parts[1]
				}

				done := make(chan error, 1)
				var ev listenerEvent
				switch cmd {
				case string(resumeCmdEvent):
					ev = listenerEvent{Type: resumeCmdEvent, Details: source, Done: done}
				case string(idleCmdEvent):
					ev = listenerEvent{Type: idleCmdEvent, Details: source, Done: done}
				case string(pingCmdEvent):
					ev = listenerEvent{Type: pingCmdEvent, Done: done}
				default:
					slog.Warn("command listener: got unknown command", "command", msg)
					return
				}

				events <- ev

				if err := <-done; err != nil {
					_, _ = fmt.Fprintf(conn, "ERROR: %v", err)
				} else {
					_, _ = conn.Write([]byte("OK"))
				}
			}()
		}
	}
}

// parseDisplayEvent splits the event string and returns what type of event it is.
func parseDisplayEvent(line string) (listenerEvent, error) {
	parts := strings.SplitN(line, ">>", 2)
	if len(parts) != 2 {
		return listenerEvent{}, fmt.Errorf("invalid event: %q", line)
	}

	ev := &listenerEvent{
		Type: displayUnknownEvent,
	}

	if et, ok := displayEvents[parts[0]]; ok {
		ev.Type = et
		ev.Details = parts[1]
	}

	return *ev, nil
}
