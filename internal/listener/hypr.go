package listener

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

// We are only actively filtering for the v2 monitor events as to not double up (since hyprland
// sends both a "v1" (monitoradded or monitorremoved) but it's expected that v2 is deprecated and just
// replaces the original, so this will probably change.
var monitorEvents = map[string]EventType{
	"monitoraddedv2":   DisplayAddEvent,
	"monitorremovedv2": DisplayRemoveEvent,
}

// ListenHyprctl listens for hyprctl events and sends an event if it is a monitor add or removal.
func (l *Listener) ListenHyprctl(ctx context.Context, events chan<- Event) error {
	var lastEvent Event
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

			if ev.Type == DisplayUnknownEvent {
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

// parseDisplayEvent splits the event string and returns what type of event it is.
func parseDisplayEvent(line string) (Event, error) {
	parts := strings.SplitN(line, ">>", 2)
	if len(parts) != 2 {
		return Event{}, fmt.Errorf("invalid event: %q", line)
	}

	ev := &Event{
		Type: DisplayUnknownEvent,
	}

	if et, ok := monitorEvents[parts[0]]; ok {
		ev.Type = et
		ev.Details = parts[1]
	}

	return *ev, nil
}
