package app

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/dsrosen6/hyprlaptop/internal/listener"
)

func SendLidCommand() error {
	return sendCommand(string(listener.LidSwitchEvent))
}

func SendWakeCommand() error {
	return sendCommand(string(listener.IdleWakeEvent))
}

func sendCommand(msg string) error {
	sock := filepath.Join(os.TempDir(), listener.CommandSockName)

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return fmt.Errorf("command listener not running")
	}

	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("Error closing socket connection: %v\n", err)
		}
	}()

	if _, err = conn.Write([]byte(msg)); err != nil {
		return fmt.Errorf("writing message '%s' to socket: %w", msg, err)
	}

	return nil
}
