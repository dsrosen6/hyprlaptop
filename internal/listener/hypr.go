package listener

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strings"
)

var monitorEvents = map[string]EventType{
	"monitoraddedv2":   DisplayAddEvent,
	"monitorremovedv2": DisplayRemoveEvent,
}

func (l *Listener) ListenHyprctl(ctx context.Context, events chan<- Event) error {
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

			events <- ev
		}
	}

	if err := scn.Err(); err != nil {
		return fmt.Errorf("error scanning: %w", err)
	}

	return nil
}

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
