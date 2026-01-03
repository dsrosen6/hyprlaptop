package main

// labeledMonitor helps tie a user-declared label to a monitor received from Hyprland.
// This prevents the updater from using partially-filled monitors from the config, and instead
// uses the up-to-date details given by Hyprland itself.
type labeledMonitor struct {
	Label   string
	Monitor monitor
}

// labelLookup is a helper struct which assists in the lookup of labeled monitors, and
// confirmation that a legitimate monitor exists for a given label.
type labelLookup struct {
	monitors []labeledMonitor
	confirm  map[string]bool
}

// newLabelLookup creates a labelLookup with the user's config monitors and current app state monitors,
// which were fetched from Hyprland.
func (a *app) newLabelLookup() labelLookup {
	return newLabelLookup(a.cfg.Monitors, a.currentState.Monitors)
}

// newLabelLookup creates a labelLookup from the user's config and the monitors fetched from Hyprland.
func newLabelLookup(cfgMtrs monitorConfigMap, hyprMtrs []monitor) labelLookup {
	lm := matchMonitorsToLabels(cfgMtrs, hyprMtrs)
	confirm := make(map[string]bool, len(lm))
	for _, l := range lm {
		confirm[l.Label] = true
	}

	return labelLookup{
		monitors: lm,
		confirm:  confirm,
	}
}

// matchMonitorsToLabels cycles through each declared monitor in the config and matches them to a
// monitor fetched from Hyprland.
func matchMonitorsToLabels(cfgMtrs monitorConfigMap, hyprMtrs []monitor) []labeledMonitor {
	var labeled []labeledMonitor
	for label, cfg := range cfgMtrs {
		for _, hm := range hyprMtrs {
			if matchesIdentifiers(hm, cfg.Identifiers) {
				labeled = append(labeled, labeledMonitor{
					Label:   label,
					Monitor: hm,
				})
				break
			}
		}
	}

	return labeled
}

// matchesIdentifiers verifies if a user-provided set of monitor identifiers has a match
// for a monitor fetched from Hyprland. It only checks non-empty fields in the identifier set,
// since the user may provide only name, only description, or some other combination.
func matchesIdentifiers(hm monitor, ident monitorIdentifiers) bool {
	if ident.Name == "" && ident.Description == "" &&
		ident.Make == "" && ident.Model == "" {
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

	if ident.Make != "" {
		if ident.Make != hm.Make {
			return false
		}
	}

	if ident.Model != "" {
		if ident.Model != hm.Model {
			return false
		}
	}

	return true
}
