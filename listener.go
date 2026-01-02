package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/godbus/dbus/v5"
)

type (
	listener struct {
		hctlSocketConn *hyprSocketConn
		dbusConn       *dbus.Conn
		cfgPath        string
	}

	listenerEvent struct {
		Type    eventType
		Details string
	}

	eventType string
)

// We are only actively filtering for the v2 monitor events as to not double up (since hyprland
// sends both a "v1" (monitoradded or monitorremoved) but it's expected that v2 is deprecated and just
// replaces the original, so this will probably change.
var monitorEvents = map[string]eventType{
	"monitoraddedv2":   displayAddEvent,
	"monitorremovedv2": displayRemoveEvent,
}

const (
	configUpdatedEvent  eventType = "CONFIG_UPDATED"
	displayInitialEvent eventType = "DISPLAY_INITIAL"
	displayAddEvent     eventType = "DISPLAY_ADDED"
	displayRemoveEvent  eventType = "DISPLAY_REMOVED"
	displayUnknownEvent eventType = "DISLAY_UNKNOWN_EVENT"
	lidSwitchEvent      eventType = "LID_SWITCH"
	powerChangedEvent   eventType = "POWER_CHANGED"
)

func newListener(hs *hyprSocketConn, dc *dbus.Conn, cfgPath string) (*listener, error) {
	return &listener{
		hctlSocketConn: hs,
		dbusConn:       dc,
		cfgPath:        cfgPath,
	}, nil
}

func (l *listener) listen(ctx context.Context, events chan<- listenerEvent) error {
	errc := make(chan error, 1)
	go func() {
		slog.Info("listening for hyprland events")
		if err := l.listenHyprctl(ctx, events); err != nil {
			errc <- fmt.Errorf("hyprland listener: %w", err)
		}
	}()

	go func() {
		slog.Info("listening for config changes")
		if err := l.listenConfigChanges(ctx, events); err != nil {
			errc <- fmt.Errorf("config listener: %w", err)
		}
	}()

	go func() {
		slog.Info("listening for lid events")
		if err := l.listenLidEvents(ctx, events); err != nil {
			errc <- fmt.Errorf("lid listener: %w", err)
		}
	}()

	go func() {
		slog.Info("listening for power events")
		if err := l.listenPowerEvents(ctx, events); err != nil {
			errc <- fmt.Errorf("power listener: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

// listenHyprctl listens for hyprctl events and sends an event if it is a monitor add or removal.
func (l *listener) listenHyprctl(ctx context.Context, events chan<- listenerEvent) error {
	// emit initial event so app queries monitors
	select {
	case events <- listenerEvent{Type: displayInitialEvent}:
		slog.Info("sent initial display event")
	case <-ctx.Done():
		return ctx.Err()
	}

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

	return nil
}

// listenConfigChanges changes listens for changes in the config file; if a change is detected,
// hyprlaptop performs a live reload.
func (l *listener) listenConfigChanges(ctx context.Context, events chan<- listenerEvent) error {
	var lastHash [32]byte

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating config file watcher: %w", err)
	}
	slog.Debug("config watcher: fsnotify watcher created")

	defer func() {
		if err := w.Close(); err != nil {
			slog.Error("closing config file watcher", "error", err)
		}
	}()

	dir := filepath.Dir(l.cfgPath)
	err = w.Add(dir)
	if err != nil {
		return fmt.Errorf("adding config directory to watcher: %w", err)
	}
	slog.Debug("config watcher: fsnotify watch list", "list", w.WatchList())

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				h, err := fileHash(l.cfgPath)
				if err != nil {
					continue
				}

				if h == lastHash {
					slog.Debug("config watcher: received identical hash for file update, no changes needed")
					continue
				}

				lastHash = h

				slog.Debug("fsnotify: file modified", "file", event.Name)
				events <- listenerEvent{
					Type:    configUpdatedEvent,
					Details: l.cfgPath,
				}
			}

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("config watcher fsnotify error: %w", err)
		}
	}
}

func (l *listener) listenLidEvents(ctx context.Context, events chan<- listenerEvent) error {
	lidListener := newLidListener(l.dbusConn)

	go func() {
		if err := lidListener.run(ctx); err != nil && err != context.Canceled {
			slog.Error("lid listener stopped", "error", err)
		}
	}()

	for lidEvent := range lidListener.events {
		select {
		case events <- listenerEvent{Type: lidSwitchEvent, Details: lidEvent.State.string()}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (l *listener) listenPowerEvents(ctx context.Context, events chan<- listenerEvent) error {
	powerListener := newPowerListener(l.dbusConn)

	go func() {
		if err := powerListener.run(ctx); err != nil && err != context.Canceled {
			slog.Error("power listener stopped", "error", err)
		}
	}()

	for powerEvent := range powerListener.events {
		select {
		case events <- listenerEvent{Type: powerChangedEvent, Details: powerEvent.State.string()}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
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

	if et, ok := monitorEvents[parts[0]]; ok {
		ev.Type = et
		ev.Details = parts[1]
	}

	return *ev, nil
}

func fileHash(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}
