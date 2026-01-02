package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/godbus/dbus/v5"
)

type (
	lidListener struct {
		conn    *dbus.Conn
		events  chan lidEvent
		signals chan *dbus.Signal
	}

	powerListener struct {
		conn    *dbus.Conn
		events  chan powerEvent
		signals chan *dbus.Signal
	}

	powerEvent struct {
		State powerState
	}

	lidEvent struct {
		State lidState
	}

	powerState int
	lidState   int
)

const (
	upowerOnBatProperty    = "OnBattery"
	upowerDest             = "org.freedesktop.UPower"
	upowerPath             = "/org/freedesktop/UPower"
	upowerMatchIfc         = "org.freedesktop.DBus.Properties"
	upowerMatchMbr         = "PropertiesChanged"
	upowerMethod           = "org.freedesktop.DBus.Properties.Get"
	upowerProperty         = "LidIsClosed"
	powerStateUnknownStr   = "unknown"
	powerStateOnBatteryStr = "battery"
	powerStateOnACStr      = "ac"
	lidStateUnknownStr     = "unknown"
	lidStateOpenedStr      = "opened"
	lidStateClosedStr      = "closed"
)

const (
	lidStateUnknown lidState = iota
	lidStateOpened
	lidStateClosed
)

const (
	powerStateUnknown powerState = iota
	powerStateOnBattery
	powerStateOnAC
)

func newLidListener(conn *dbus.Conn) *lidListener {
	return &lidListener{
		conn:    conn,
		events:  make(chan lidEvent, 10),
		signals: make(chan *dbus.Signal, 10),
	}
}

func newPowerListener(conn *dbus.Conn) *powerListener {
	return &powerListener{
		conn:    conn,
		events:  make(chan powerEvent, 10),
		signals: make(chan *dbus.Signal, 10),
	}
}

func (l *lidListener) run(ctx context.Context) error {
	defer close(l.events)
	defer l.conn.RemoveSignal(l.signals)
	if err := l.start(ctx); err != nil {
		return err
	}

	initialState, err := l.getCurrentState(ctx)
	if err != nil {
		return fmt.Errorf("getting initial lid state: %w", err)
	}

	select {
	case l.events <- lidEvent{State: initialState}:
	case <-ctx.Done():
		return ctx.Err()
	}

	lastState := initialState

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

func (l *lidListener) start(ctx context.Context) error {
	if err := l.conn.AddMatchSignalContext(
		ctx, dbus.WithMatchInterface(upowerMatchIfc), dbus.WithMatchMember(upowerMatchMbr),
		dbus.WithMatchObjectPath(dbus.ObjectPath(upowerPath)),
	); err != nil {
		return fmt.Errorf("failed to add dbus match rule: %w", err)
	}

	l.conn.Signal(l.signals)
	return nil
}

func (l *lidListener) getCurrentState(ctx context.Context) (lidState, error) {
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

func (l *lidListener) shouldHandleSignal(sig *dbus.Signal) bool {
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

func (p *powerListener) run(ctx context.Context) error {
	defer close(p.events)
	defer p.conn.RemoveSignal(p.signals)
	if err := p.start(ctx); err != nil {
		return err
	}

	initialState, err := p.getCurrentState(ctx)
	if err != nil {
		return fmt.Errorf("getting initial lid state: %w", err)
	}

	select {
	case p.events <- powerEvent{State: initialState}:
	case <-ctx.Done():
		return ctx.Err()
	}

	lastState := initialState

	for {
		select {
		case sig, ok := <-p.signals:
			if !ok {
				return fmt.Errorf("signals channel closed")
			}

			if !p.shouldHandleSignal(sig) {
				continue
			}

			currentState, err := p.getCurrentState(ctx)
			if err != nil {
				return fmt.Errorf("failed to get current lid state: %w", err)
			}

			if currentState != lastState {
				select {
				case p.events <- powerEvent{State: currentState}:
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

func (p *powerListener) start(ctx context.Context) error {
	if err := p.conn.AddMatchSignalContext(
		ctx, dbus.WithMatchInterface(upowerMatchIfc), dbus.WithMatchMember(upowerMatchMbr),
		dbus.WithMatchObjectPath(dbus.ObjectPath(upowerPath)),
	); err != nil {
		return fmt.Errorf("failed to add dbus match rule: %w", err)
	}

	p.conn.Signal(p.signals)
	return nil
}

func (p *powerListener) getCurrentState(ctx context.Context) (powerState, error) {
	obj := p.conn.Object(upowerDest, upowerPath)
	var result dbus.Variant
	if err := obj.CallWithContext(ctx, upowerMethod, 0, upowerDest, upowerOnBatProperty).Store(&result); err != nil {
		return powerStateUnknown, err
	}

	if onBat, ok := result.Value().(bool); ok {
		if onBat {
			return powerStateOnBattery, nil
		}
		return powerStateOnAC, nil
	}

	return powerStateUnknown, fmt.Errorf("unexpected type for OnBattery")
}

func (p *powerListener) shouldHandleSignal(sig *dbus.Signal) bool {
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

func (ls lidState) string() string {
	switch ls {
	case lidStateOpened:
		return lidStateOpenedStr
	case lidStateClosed:
		return lidStateClosedStr
	default:
		return lidStateUnknownStr
	}
}

func (ps powerState) string() string {
	switch ps {
	case powerStateOnAC:
		return powerStateOnACStr
	case powerStateOnBattery:
		return powerStateOnBatteryStr
	default:
		return powerStateUnknownStr
	}
}

func parseLidState(s string) lidState {
	switch s {
	case lidStateOpenedStr:
		return lidStateOpened
	case lidStateClosedStr:
		return lidStateClosed
	default:
		return lidStateUnknown
	}
}

func parsePowerState(s string) powerState {
	switch s {
	case powerStateOnACStr:
		return powerStateOnAC
	case powerStateOnBatteryStr:
		return powerStateOnBattery
	default:
		return powerStateUnknown
	}
}
