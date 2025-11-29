package main

import (
	"fmt"
	"strings"
)

type baseEvent struct {
	name    string
	payload string
}

func extractMonitorName(payload string) (string, error) {
	parts := strings.Split(payload, ",")
	if len(parts) != 3 {
		return "", fmt.Errorf("bad monitorv2 event: %q", payload)
	}

	return parts[1], nil
}

func parseBaseEvent(line string) (*baseEvent, error) {
	parts := strings.SplitN(line, ">>", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid event: %q", line)
	}

	return &baseEvent{
		name:    parts[0],
		payload: parts[1],
	}, nil
}
