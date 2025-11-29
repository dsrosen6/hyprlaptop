package main

import (
	"log/slog"
	"os"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

func main() {
	panic(run())
}

func run() error {
	if os.Getenv("DEBUG") == "true" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	conn, err := hypr.NewConn()
	if err != nil {
		return err
	}

	if err := conn.Listen(); err != nil {
		return err
	}

	return nil
}
