package app

import (
	"fmt"
	"os"
	"strings"
)

const (
	lidStateFile = "/proc/acpi/button/lid/LID/state"
)

type lidState string

const (
	lidStateUnknown lidState = "Unknown"
	lidStateOpen    lidState = "Open"
	lidStateClosed  lidState = "Closed"
)

func getLidState() (lidState, error) {
	b, err := os.ReadFile(lidStateFile)
	if err != nil {
		return lidStateUnknown, fmt.Errorf("reading lid state file: %w", err)
	}
	s := strings.ToLower(string(b))

	switch {
	case strings.Contains(s, "open"):
		return lidStateOpen, nil
	case strings.Contains(s, "closed"):
		return lidStateClosed, nil
	default:
		return lidStateUnknown, nil
	}
}
