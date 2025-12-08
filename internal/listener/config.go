package listener

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func (l *Listener) listenForConfigChanges(ctx context.Context, events chan<- Event) error {
	var lastHash [32]byte

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating config file watcher: %w", err)
	}
	slog.Debug("config watcher: fsnotify watcher created")

	defer func() {
		if err := w.Close(); err != nil {
			slog.Error("closing config file watcher", "error", err)
		}
	}()

	dir := filepath.Dir(l.cfgPath)
	err = w.Add(dir)
	if err != nil {
		return fmt.Errorf("adding config directory to watcher: %w", err)
	}
	slog.Debug("config watcher: fsnotify watch list", "list", w.WatchList())

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}

			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				h, err := fileHash(l.cfgPath)
				if err != nil {
					continue
				}

				if h == lastHash {
					slog.Debug("config watcher: received identical hash for file update, no changes needed")
					continue
				}

				lastHash = h

				slog.Debug("fsnotify: file modified", "file", event.Name)
				events <- Event{
					Type:    ConfigUpdatedEvent,
					Details: l.cfgPath,
				}
			}

		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("config watcher fsnotify error: %w", err)
		}
	}
}

func fileHash(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}
