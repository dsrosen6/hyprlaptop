package main

import (
	"log/slog"
	"strings"
)

type State struct {
	Monitors   monitorMap
	LidState   lidState
	PowerState powerState
}

func (s *State) Ready() bool {
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
