package app

import (
	"log/slog"
	"os/exec"
	"time"
)

func (a *App) runUpdater() (bool, error) {
	a.updating = true
	defer func() {
		a.updating = false
	}()

	if a.mode == modeIdle {
		return a.handleIdleCmd()
	}

	s := a.status()
	lg := slog.Default().With(
		slog.String("mode", a.mode.string()),
		slog.String("status", s.string()),
	)

	switch s {
	case statusDockedOpened, statusOnlyLaptopOpened:
		switch a.laptopIsEnabled() {
		case true:
			lg.Debug("[UPDATER]laptop display already enabled; no action needed")
		case false:
			lg.Info("[UPDATER]enabling laptop display")
			return true, a.hctl.EnableOrUpdateMonitor(a.laptopDisplay)
		}

	case statusOnlyLaptopClosed:
		if a.Config.SuspendClosed {
			lg.Info("[UPDATER]suspending machine")
			return a.handleIdleCmd()
		}

		switch a.laptopIsEnabled() {
		case true:
			lg.Debug("[UPDATER]laptop display already enabled; no action needed")
		case false:
			lg.Info("[UPDATER]enabling laptop display")
			return true, a.hctl.EnableOrUpdateMonitor(a.laptopDisplay)
		}

	case statusDockedClosed:
		switch a.laptopIsEnabled() {
		case true:
			lg.Info("[UPDATER]disabling laptop display")
			return true, a.hctl.DisableMonitor(a.laptopDisplay)
		case false:
			lg.Debug("[UPDATER]laptop display already disabled; no action needed")
		}
	default:
		lg.Info("[UPDATER]unknown status; doing nothing")
	}

	return false, nil
}

func (a *App) handleIdleCmd() (bool, error) {
	if a.Config.LockOnIdle {
		lockCmd := a.Config.LockCmd
		if lockCmd == "" {
			lockCmd = DefaultLockCmd
		}
		lockDelay := a.Config.LockDelay
		if lockDelay <= 0 {
			lockDelay = defaultLockDelay
		}
		slog.Info("[UPDATER/IDLE CMD]running lock command", "command", lockCmd)
		if err := exec.Command("sh", "-c", lockCmd).Start(); err != nil {
			slog.Error("[UPDATER/IDLE CMD]issue running lock command", "error", err)
		}
		time.Sleep(time.Duration(lockDelay) * time.Second)
	}

	changed := false
	if !a.laptopIsEnabled() {
		slog.Info("[UPDATER/IDLE CMD]enabling laptop display")
		if err := a.hctl.EnableOrUpdateMonitor(a.laptopDisplay); err != nil {
			slog.Error("[UPDATER/IDLE CMD]issue enabling laptop display", "error", err)
		}
		changed = true
	} else {
		slog.Info("[UPDATER/IDLE CMD]laptop display already enabled")
	}

	if a.Config.SuspendIdle {
		sd := a.Config.SuspendDelay
		if sd <= 0 {
			sd = defaultSuspendDelay
		}
		time.Sleep(time.Duration(sd) * time.Second)
		slog.Info("[UPDATER/IDLE CMD]suspending on idle enabled; suspending")
		return changed, systemctlSuspend()
	}
	slog.Info("[UPDATER/IDLE CMD]suspending on idle disabled; doing nothing")
	return changed, nil
}

func (a *App) runPostHooks(changed bool) {
	for _, hook := range a.Config.PostUpdateHooks {
		if hook.OnStatusChange && !changed {
			continue
		}
		cmd := hook.Command
		if a.Config.SequentialHooks {
			slog.Debug("running post-hook", "command", cmd)
			if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
				slog.Error("post-hook failed", "command", cmd, "error", err)
			}
		} else {
			go func() {
				slog.Debug("running post-hook", "command", cmd)
				if err := exec.Command("sh", "-c", cmd).Run(); err != nil {
					slog.Error("post-hook failed", "command", cmd, "error", err)
				}
			}()
		}
	}
}

func systemctlSuspend() error {
	cmd := exec.Command("systemctl", "suspend")
	return cmd.Run()
}
