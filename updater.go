package main

import (
	"fmt"
	"log/slog"
	"reflect"
	"time"
)

func (a *app) runUpdater() error {
	a.updating = true
	defer func() {
		a.lastUpdateEnd = time.Now()
		a.updating = false
	}()

	lookup := a.newLabelLookup()
	matched := a.getMatchingProfile(lookup)
	if matched == nil {
		slog.Info("no profile match found")
		return nil
	}

	params := a.updateParamsFromProfile(matched, lookup)
	logUpdateParams(*params)
	if len(params.enableOrUpdate) == 0 && len(params.disable) == 0 {
		slog.Info("found profile match; no changes needed", "profile", matched.Name)
		return nil
	}

	slog.Info("found profile match; applying updates", "profile", matched.Name)
	if err := a.hctl.bulkUpdateMonitors(params); err != nil {
		return fmt.Errorf("bulk updating monitors: %w", err)
	}

	return nil
}

func (a *app) updateParamsFromProfile(p *profile, lookup labelLookup) *monitorUpdateParams {
	var toUpdate, toDisable, noChanges []monitor
	seenNames := newSet[string]()

	for _, s := range p.MonitorStates {
		lm, ok := lookup[s.Label]
		if !ok {
			continue
		}
		seenNames.add(lm.Monitor.Name)

		if !lm.CurrentlyEnabled {
			if s.Disable {
				slog.Debug("update param builder: no monitor found for label, but marked for disable, no action needed", "label", s.Label)
				continue
			}
		}

		if s.Disable {
			toDisable = append(toDisable, lm.Monitor)
			continue
		}

		if s.Preset == nil {
			// no preset provided, just assume it stays the same
			noChanges = append(toUpdate, lm.Monitor)
			continue
		}

		pr, ok := a.cfg.Monitors[s.Label].Presets[*s.Preset]
		if !ok {
			slog.Warn("update param builder: provided preset doesn't exist", "preset", s.Preset)
			continue
		}

		// check if changes are needed
		if reflect.DeepEqual(pr, lm.Monitor.monitorSettings) {
			noChanges = append(noChanges, lm.Monitor)
			continue
		}

		m := monitor{
			monitorIdentifiers: lm.Monitor.monitorIdentifiers,
			monitorSettings:    pr,
		}

		toUpdate = append(toUpdate, m)
	}

	if p.DisableUndeclared {
		for _, m := range a.currentState.Monitors {
			if seenNames.contains(m.Name) {
				toDisable = append(toDisable, m)
			}
		}
	}

	return newMonitorUpdateParams(toUpdate, toDisable, noChanges)
}

func logUpdateParams(params monitorUpdateParams) {
	for _, m := range params.enableOrUpdate {
		slog.Debug("will update monitor", "name", m.Name, "desc", m.Description)
	}

	for _, m := range params.disable {
		slog.Debug("will disable monitor", "name", m.Name, "desc", m.Description)
	}

	for _, m := range params.noChanges {
		slog.Debug("no changes needed for monitor", "name", m.Name, "desc", m.Description)
	}
}
