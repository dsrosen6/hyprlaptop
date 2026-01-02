package app

import (
	"github.com/dsrosen6/hyprlaptop/internal/config"
	"github.com/dsrosen6/hyprlaptop/internal/hypr"
	"github.com/dsrosen6/hyprlaptop/internal/power"
)

type App struct {
	Hctl     *hypr.HyprctlClient
	Cfg      *config.Config
	Profiles []Profile
	State    *State
}

type State struct {
	Monitors   hypr.MonitorMap
	LidState   power.LidState
	PowerState power.PowerState
}

func NewApp(cfg *config.Config, hc *hypr.HyprctlClient) *App {
	return &App{
		Hctl:  hc,
		Cfg:   cfg,
		State: &State{},
	}
}

func (a *App) RunUpdater() error {
	return nil
}

func (a *App) PowerStatesReady() bool {
	return a.State != nil && a.State.LidState != power.LidStateUnknown && a.State.PowerState != power.PowerStateUnknown
}
