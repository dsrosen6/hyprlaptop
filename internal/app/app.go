package app

import (
	"fmt"
	"strings"

	"github.com/dsrosen6/hyprlaptop/internal/config"
	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

type App struct {
	Hctl *hypr.HyprctlClient
	Cfg  *config.Config
}

func NewApp(cfg *config.Config, hc *hypr.HyprctlClient) *App {
	return &App{
		Hctl: hc,
		Cfg:  cfg,
	}
}

func (a *App) SaveCurrentDisplays(laptop string) error {
	displays, err := a.Hctl.ListMonitors()
	if err != nil {
		return fmt.Errorf("listing displays via hyprctl: %w", err)
	}

	var lm *hypr.Monitor
	if laptop == "" {
		for _, m := range displays {
			if strings.Contains(m.Name, "eDP") {
				lm = &m
			}
		}
	} else {
		l, ok := displays[laptop]
		if ok {
			lm = &l
		}
	}

	if lm == nil {
		return fmt.Errorf("display '%s' not found", laptop)
	}

	externals := map[string]hypr.Monitor{}
	for _, m := range displays {
		if m.Name != lm.Name {
			externals[m.Name] = m
		}
	}

	a.Cfg.LaptopDisplay = *lm
	a.Cfg.ExternalDisplays = externals

	if err := a.Cfg.Write(); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
