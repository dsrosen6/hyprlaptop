package main

type labeledMonitor struct {
	Label   string
	Monitor monitor
}

func (a *app) matchMonitorsToLabels() []labeledMonitor {
	var labeled []labeledMonitor
	for label, cfg := range a.monitors {
		for _, hm := range a.state.Monitors {
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
