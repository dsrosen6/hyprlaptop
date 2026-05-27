package app

import (
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	configReloadDelay  = 100 * time.Millisecond
	DefaultLockCmd     = "pidof hyprlock || hyprlock"
	defaultLockDelay   = 1
	defaultSuspendDelay = 1
)

type Config struct {
	Debug           bool       `mapstructure:"debug"`
	Laptop          string     `mapstructure:"laptop"`
	LockOnIdle      bool       `mapstructure:"lock-on-idle"`
	LockCmd         string     `mapstructure:"lock-cmd"`
	LockDelay       int        `mapstructure:"lock-delay"`
	SuspendDelay    int        `mapstructure:"suspend-delay"`
	SuspendIdle     bool       `mapstructure:"suspend-idle"`
	SuspendClosed   bool       `mapstructure:"suspend-closed"`
	PostUpdateHooks []PostHook `mapstructure:"post-hooks"`
	SequentialHooks bool       `mapstructure:"sequential-hooks"`
	SettleWindow    int        `mapstructure:"settle-window"`
}

type PostHook struct {
	Command        string `mapstructure:"command"`
	OnStatusChange bool   `mapstructure:"on-status-change"`
}

// onConfigChange handles live updates when a config file change is detected.
func (a *App) onConfigChange(e fsnotify.Event) {
	if a.configReloadTimer != nil {
		a.configReloadTimer.Stop()
	}

	a.configReloadTimer = time.AfterFunc(configReloadDelay, func() {
		var newCfg Config
		if err := viper.Unmarshal(&newCfg); err != nil {
			slog.Error("reloading config", "error", err)
			return
		}
		select {
		case a.listener.configCh <- newCfg:
			slog.Info("config reloaded", "config", newCfg)
		default:
		}
	})
}
