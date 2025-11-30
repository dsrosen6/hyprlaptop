package main

import (
	"errors"
	"fmt"
)

// Monitor matches the output of 'hyprctl monitors', and is also used for config.
type Monitor struct {
	ID          int64   `json:"id,omitempty"`
	Name        string  `json:"name,omitempty"`
	Width       int64   `json:"width,omitempty"`
	Height      int64   `json:"height,omitempty"`
	RefreshRate float64 `json:"refreshRate,omitempty"`
	X           int64   `json:"x,omitempty"`
	Y           int64   `json:"y,omitempty"`
	Scale       float64 `json:"scale,omitempty"`
}

func (a *app) saveCurrentMonitors(laptopMtr string) error {
	if laptopMtr == "" {
		return errors.New("no laptop monitor name provided")
	}

	monitors, err := a.hc.listMonitors()
	if err != nil {
		return fmt.Errorf("fetching all current monitors via hyprctl: %w", err)
	}

	lm, valid := monitors[laptopMtr]
	if !valid {
		return fmt.Errorf("setting laptop monitor: monitor '%s' not found", laptopMtr)
	}

	externals := map[string]Monitor{}
	for _, m := range monitors {
		if m.Name != lm.Name {
			externals[m.Name] = m
		}
	}

	a.cfg.LaptopMonitor = lm
	a.cfg.ExternalMonitors = externals

	if err := a.cfg.write(); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func (a *app) setLaptopMonitor(name string) error {
	if name == "" {
		return errors.New("no monitor name provided")
	}

	monitors, err := a.hc.listMonitors()
	if err != nil {
		return fmt.Errorf("fetching monitors from hyprctl: %w", err)
	}

	m, valid := monitors[name]
	if !valid {
		return fmt.Errorf("monitor '%s' not found", name)
	}

	a.cfg.LaptopMonitor = m
	return a.cfg.write()
}

func (h *hyprctlClient) listMonitors() (map[string]Monitor, error) {
	var monitors []Monitor
	if err := h.runCommandWithUnmarshal([]string{"monitors"}, &monitors); err != nil {
		return nil, err
	}

	mm := make(map[string]Monitor, len(monitors))
	for _, m := range monitors {
		mm[m.Name] = m
	}

	return mm, nil
}
