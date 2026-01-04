package main

import "log/slog"

// labeledMonitor helps tie a user-declared label to a monitor received from Hyprland.
// This prevents the updater from using partially-filled monitors from the config, and instead
// uses the up-to-date details given by Hyprland itself.
type labeledMonitor struct {
	Label            string
	Monitor          monitor
	CurrentlyEnabled bool
}

// labelLookup is a helper struct which assists in the lookup of labeled monitors, and
// confirmation that a legitimate monitor exists for a given label.
type labelLookup map[string]labeledMonitor

// newLabelLookup creates a labelLookup with the user's config monitors and current app state monitors,
// which were fetched from Hyprland.
func (a *app) newLabelLookup() labelLookup {
	return newLabelLookup(a.cfg.Monitors, a.currentState.Monitors)
}

// newLabelLookup creates a labelLookup from the user's config and the monitors fetched from Hyprland.
func newLabelLookup(cfgMtrs monitorConfigMap, hyprMtrs []monitor) labelLookup {
	lookup := make(labelLookup)
	for label, cfg := range cfgMtrs {
		found := false
		// try hyprland monitors first since it will have more details
		for _, hm := range hyprMtrs {
			if matchesIdentifiers(hm, cfg.Identifiers) {
				slog.Debug("newLabelLookup: found hyprland match", "label", label, "name", hm.Name)
				lookup[label] = labeledMonitor{
					Label:            label,
					Monitor:          hm,
					CurrentlyEnabled: true,
				}
				found = true
				break
			}
		}

		// if not found, use values from config. This will usually be the case for
		// monitors that have been disabled by a profile; it won't show on the hyprland output.
		if !found {
			slog.Debug("newLabelLookup: adding monitor from config", "label", label, "name", cfg.Identifiers.Name, "desc", cfg.Identifiers.Description)
			lookup[label] = labeledMonitor{
				Label: label,
				Monitor: monitor{
					monitorIdentifiers: cfg.Identifiers,
					// settings to be filled from preset later
				},
			}
		}
	}

	return lookup
}

// matchesIdentifiers verifies if a user-provided set of monitor identifiers has a match
// for a monitor fetched from Hyprland. It only checks non-empty fields in the identifier set,
// since the user may provide only name, only description, or some other combination.
func matchesIdentifiers(hm monitor, ident monitorIdentifiers) bool {
	if ident.Name == "" && ident.Description == "" {
		return false
	}

	if ident.Name != "" {
		if ident.Name != hm.Name {
			return false
		}
	}

	if ident.Description != "" {
		if ident.Description != hm.Description {
			return false
		}
	}

	return true
}
