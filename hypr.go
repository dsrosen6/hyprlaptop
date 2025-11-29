package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const (
	binaryName       = "hyprctl"
	unknownReqOutput = "unknown request"
)

type client struct {
	BinaryPath string
}

var errUnknownRequest = errors.New(unknownReqOutput)

func newClient() (*client, error) {
	bp, err := exec.LookPath(binaryName)
	if err != nil {
		return nil, fmt.Errorf("finding full hyprctl binary path: %w", err)
	}

	return &client{
		BinaryPath: bp,
	}, nil
}

func (c *client) runCommandWithUnmarshal(args []string, v any) error {
	a := append([]string{"-j"}, args...)
	out, err := c.runCommand(a)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(out, v); err != nil {
		return fmt.Errorf("unmarshaling json: %w", err)
	}

	return nil
}

func (c *client) runCommand(args []string) ([]byte, error) {
	cmd := exec.Command(c.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running command: %w", err)
	}

	out := stdout.Bytes()
	errStr := strings.TrimSpace(stderr.String())
	if errStr != "" {
		return nil, errors.New(errStr)
	}

	return out, checkForErr(string(out))
}

func checkForErr(out string) error {
	out = strings.TrimSpace(out)
	switch out {
	case unknownReqOutput:
		return errUnknownRequest
	default:
		return nil
	}
}
