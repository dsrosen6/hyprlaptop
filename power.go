package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/godbus/dbus/v5"
)

type (
	powerHandler struct {
		conn    *dbus.Conn
		events  chan powerEvent
		signals chan *dbus.Signal
	}

	powerEvent struct {
		State powerState
	}

	powerState string
)

const (
	upowerOnBatProperty = "OnBattery"
	upowerDest          = "org.freedesktop.UPower"
	upowerPath          = "/org/freedesktop/UPower"
	upowerMatchIfc      = "org.freedesktop.DBus.Properties"
	upowerMatchMbr      = "PropertiesChanged"
	upowerMethod        = "org.freedesktop.DBus.Properties.Get"
	upowerProperty      = "LidIsClosed"

	powerStateUnknown   powerState = "unknown"
	powerStateOnBattery powerState = "battery"
	powerStateOnAC      powerState = "ac"
)

func newPowerHandler(conn *dbus.Conn) *powerHandler {
	return &powerHandler{
		conn:    conn,
		events:  make(chan powerEvent, 10),
		signals: make(chan *dbus.Signal, 10),
	}
}

func (p *powerHandler) listenForChanges(ctx context.Context) error {
	defer close(p.events)
	defer p.conn.RemoveSignal(p.signals)
	if err := p.startDbus(ctx); err != nil {
		return err
	}

	var lastState powerState
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

func (p *powerHandler) startDbus(ctx context.Context) error {
	if err := p.conn.AddMatchSignalContext(
		ctx, dbus.WithMatchInterface(upowerMatchIfc), dbus.WithMatchMember(upowerMatchMbr),
		dbus.WithMatchObjectPath(dbus.ObjectPath(upowerPath)),
	); err != nil {
		return fmt.Errorf("failed to add dbus match rule: %w", err)
	}

	p.conn.Signal(p.signals)
	return nil
}

func (p *powerHandler) getCurrentState(ctx context.Context) (powerState, error) {
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

func (p *powerHandler) shouldHandleSignal(sig *dbus.Signal) bool {
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

func parsePowerState(s string) powerState {
	switch powerState(s) {
	case powerStateOnBattery, powerStateOnAC:
		return powerState(s)
	default:
		return powerStateUnknown
	}
}
