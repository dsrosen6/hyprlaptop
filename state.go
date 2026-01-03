package main

import (
	"log/slog"
	"strings"
)

type state struct {
	Monitors   []monitor
	LidState   lidState
	PowerState powerState
}

func (s *state) ready() bool {
	if s == nil {
		slog.Error("state ready check", "error", "state nil")
		return false
	}

	var notReady []string
	if s.LidState == lidStateUnknown {
		notReady = append(notReady, "lid")
	}

	if s.PowerState == powerStateUnknown {
		notReady = append(notReady, "power")
	}

	if len(s.Monitors) == 0 {
		notReady = append(notReady, "monitors")
	}

	if len(notReady) > 0 {
		nr := strings.Join(notReady, ",")
		slog.Info("ready check: one or more states not ready", "states", nr)
		return false
	}

	return true
}
