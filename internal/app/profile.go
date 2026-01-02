package app

import (
	"github.com/dsrosen6/hyprlaptop/internal/config"
	"github.com/dsrosen6/hyprlaptop/internal/hypr"
	"github.com/dsrosen6/hyprlaptop/internal/power"
)

type (
	Profile struct {
		Monitors   map[string]MonitorIdentifiers `json:"monitors,omitempty"`
		LidState   *power.LidState
		PowerState *power.PowerState
	}

	MonitorIdentifiers struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		Make        *string `json:"make,omitempty"`
		Model       *string `json:"model,omitempty"`
	}
)

func (p Profile) MatchesState(s *State) bool {
}

func monitorsMatch(p MonitorIdentifiers, s hypr.Monitor) bool {
	if p.Name != nil {
		if *p.Name != s.Name {
			return false
		}
	}

	if p.Description != nil {
		if *p.Description != s.Description {
			return false
		}
	}

	if p.Make != nil {
		if *p.Make != s.Make {
			return false
		}
	}

	if p.Model != nil {
		if *p.Model != s.Model {
			return false
		}
	}

	return true
}

func cfgProfileToProfile(cfp config.Profile) Profile {
	p := &Profile{}

	if cfp.Monitors != nil {
		p.Monitors = make(map[string]MonitorIdentifiers, len(cfp.Monitors))
		for k, v := range cfp.Monitors {
			p.Monitors[k] = monitorFromConfig(v)
		}
	}

	if cfp.LidState != "" {
		ls := power.ParseLidState(cfp.LidState)
		if ls != power.LidStateUnknown {
			p.LidState = &ls
		}
	}

	if cfp.PowerState != "" {
		ps := power.ParsePowerState(cfp.PowerState)
		if ps != power.PowerStateUnknown {
			p.PowerState = &ps
		}
	}

	return *p
}

func monitorFromConfig(cm config.MonitorIdentifiers) MonitorIdentifiers {
	return MonitorIdentifiers{
		Name:        strToPtr(cm.Name),
		Description: strToPtr(cm.Description),
		Make:        strToPtr(cm.Make),
		Model:       strToPtr(cm.Model),
	}
}

func strToPtr(s string) *string {
	if s != "" {
		return &s
	}
	return nil
}
