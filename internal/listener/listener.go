package listener

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

type Listener struct {
	hctlSocketConn *hypr.SocketConn
	cfgPath        string
}

func NewListener(hctlSocketConn *hypr.SocketConn, cfgPath string) *Listener {
	return &Listener{
		hctlSocketConn: hctlSocketConn,
		cfgPath:        cfgPath,
	}
}

func ListenForEvents(ctx context.Context, cfgPath string, events chan<- Event) error {
	sc, err := hypr.NewSocketConn()
	if err != nil {
		return fmt.Errorf("creating hyprland socket connection: %w", err)
	}

	defer func() {
		if err := sc.Close(); err != nil {
			slog.Error("closing hyprland socket connection", "error", err)
		}
	}()

	l := NewListener(sc, cfgPath)
	return l.listenForEvents(ctx, events)
}

func (l *Listener) listenForEvents(ctx context.Context, events chan<- Event) error {
	errc := make(chan error, 1)
	defer func() {
		if err := l.hctlSocketConn.Close(); err != nil {
			slog.Error("closing hypr socket connection", "error", err)
		}
	}()

	go func() {
		if err := l.ListenHyprctl(ctx, events); err != nil {
			errc <- fmt.Errorf("hyprland listener: %w", err)
		}
	}()

	go func() {
		if err := l.listenForConfigChanges(ctx, events); err != nil {
			errc <- fmt.Errorf("config listener: %w", err)
		}
	}()

	go func() {
		if err := l.commandListener(ctx, events); err != nil {
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
