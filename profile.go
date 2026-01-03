package main

import (
	"log/slog"
)

type (
	profile struct {
		Name              string         `json:"name"`
		Conditions        conditions     `json:"conditions"`
		MonitorStates     []monitorState `json:"monitor_states"`
		DisableUndeclared bool           `json:"disable_undeclared_monitors"`
		valid             bool
	}

	conditions struct {
		LidState        *lidState   `json:"lid_state"`
		PowerState      *powerState `json:"power_state"`
		EnabledMonitors []string    `json:"enabled_monitors"`
	}

	monitorState struct {
		Label    string  `json:"label"`
		Disabled bool    `json:"disabled"`
		Preset   *string `json:"preset"`
	}
)

func (a *app) getMatchingProfile() *profile {
	lm := a.matchMonitorsToLabels()
	labels := make(map[string]bool)
	for _, l := range lm {
		labels[l.Label] = true
	}

	var matched *profile
	for _, p := range a.profiles {
		if a.profileMatchesState(p, labels) {
			matched = p
		}
	}

	return matched
}

func (a *app) profileMatchesState(p *profile, labels map[string]bool) bool {
	if p.Conditions.LidState != nil {
		if *p.Conditions.LidState != a.state.LidState {
			return false
		}
	}

	if p.Conditions.PowerState != nil {
		if *p.Conditions.PowerState != a.state.PowerState {
			return false
		}
	}

	for _, requiredLabel := range p.Conditions.EnabledMonitors {
		if !labels[requiredLabel] {
			return false
		}
	}

	return true
}

func (a *app) validateAllProfiles() {
	for _, p := range a.profiles {
		a.validateProfile(p)
	}
}

func (a *app) validateProfile(p *profile) {
	valid := true
	pLog := slog.Default().With(slog.String("profile_name", p.Name))
	if p.Conditions.LidState != nil {
		parsed := parseLidState(string(*p.Conditions.LidState))
		if parsed == lidStateUnknown {
			valid = false
			pLog.Warn("invalid condition: lid state")
		}
	}

	if p.Conditions.PowerState != nil {
		parsed := parsePowerState(string(*p.Conditions.PowerState))
		if parsed == powerStateUnknown {
			pLog.Warn("invalid condition: power state")
		}
	}

	for _, m := range p.Conditions.EnabledMonitors {
		if !a.validMonitorLabel(m) {
			valid = false
			pLog.Warn("invalid condition: enabled monitor", "label", m)
		}
	}

	for _, s := range p.MonitorStates {
		if !a.validMonitorLabel(s.Label) {
			valid = false
			pLog.Warn("invalid monitor state", "label", s.Label, "reason", "label not found")
			continue
		}

		if s.Preset != nil {
			if s.Disabled {
				valid = false
				pLog.Warn("invalid monitor state", "label", s.Label, "reason", "conflict: disabled set to true, but preset declared")
				continue
			}

			if !a.validMonitorPreset(s.Label, *s.Preset) {
				valid = false
				pLog.Warn("invalid monitor state", "label", s.Label, "reason", "preset not found", "preset", *s.Preset)
			}
		}
	}

	p.valid = valid
}

func (a *app) validMonitorLabel(label string) bool {
	return validMonitorLabel(a.monitors, label)
}

func (a *app) validMonitorPreset(monitor, preset string) bool {
	if !a.validMonitorLabel(monitor) {
		return false
	}

	return validMonitorPreset(a.monitors[monitor].Presets, preset)
}

func validMonitorLabel(monitors monitorConfigMap, label string) bool {
	if _, ok := monitors[label]; ok {
		return true
	}
	return false
}

func validMonitorPreset(presets monitorPresetMap, label string) bool {
	if _, ok := presets[label]; ok {
		return true
	}
	return false
}
