package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/godbus/dbus/v5"
)

type (
	lidHandler struct {
		conn    *dbus.Conn
		events  chan lidEvent
		signals chan *dbus.Signal
	}

	lidEvent struct {
		State lidState
	}

	lidState string
)

const (
	lidStateUnknown lidState = "unknown"
	lidStateOpened  lidState = "opened"
	lidStateClosed  lidState = "closed"
)

func newLidHandler(conn *dbus.Conn) *lidHandler {
	return &lidHandler{
		conn:    conn,
		events:  make(chan lidEvent, 10),
		signals: make(chan *dbus.Signal, 10),
	}
}

func (l *lidHandler) listenForChanges(ctx context.Context) error {
	defer close(l.events)
	defer l.conn.RemoveSignal(l.signals)
	if err := l.startDbus(ctx); err != nil {
		return err
	}

	var lastState lidState
	for {
		select {
		case sig, ok := <-l.signals:
			if !ok {
				return fmt.Errorf("signals channel closed")
			}

			if !l.shouldHandleSignal(sig) {
				continue
			}

			currentState, err := l.getCurrentState(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current lid state: %w", err)
			}

			if currentState != lastState {
				select {
				case l.events <- lidEvent{State: currentState}:
					lastState = currentState
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		// TODO: handle
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *lidHandler) startDbus(ctx context.Context) error {
	if err := l.conn.AddMatchSignalContext(
		ctx, dbus.WithMatchInterface(upowerMatchIfc), dbus.WithMatchMember(upowerMatchMbr),
		dbus.WithMatchObjectPath(dbus.ObjectPath(upowerPath)),
	); err != nil {
		return fmt.Errorf("failed to add dbus match rule: %w", err)
	}

	l.conn.Signal(l.signals)
	return nil
}

func (l *lidHandler) getCurrentState(ctx context.Context) (lidState, error) {
	obj := l.conn.Object(upowerDest, upowerPath)
	var result dbus.Variant
	if err := obj.CallWithContext(ctx, upowerMethod, 0, upowerDest, upowerProperty).Store(&result); err != nil {
		return lidStateUnknown, err
	}

	if closed, ok := result.Value().(bool); ok {
		if closed {
			return lidStateClosed, nil
		}
		return lidStateOpened, nil
	}

	return lidStateUnknown, fmt.Errorf("unexpected type for LidIsClosed")
}

func (l *lidHandler) shouldHandleSignal(sig *dbus.Signal) bool {
	if sig.Name != "org.freedesktop.DBus.Properties.PropertiesChanged" {
		return false
	}

	if len(sig.Body) < 2 {
		return false
	}

	if changedProps, ok := sig.Body[1].(map[string]dbus.Variant); ok {
		if _, exists := changedProps[upowerProperty]; exists {
			return true
		}
	}

	if len(sig.Body) >= 3 {
		if invalidated, ok := sig.Body[2].([]string); ok {
			if slices.Contains(invalidated, upowerProperty) {
				return true
			}
		}
	}

	return false
}

func parseLidState(s string) lidState {
	switch lidState(s) {
	case lidStateOpened, lidStateClosed:
		return lidState(s)
	default:
		return lidStateUnknown
	}
}
