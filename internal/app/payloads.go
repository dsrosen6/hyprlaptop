package app

import (
	"log/slog"
	"reflect"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
)

type displayPayload struct {
	in         hypr.Monitor
	out        hypr.Monitor
	enable     bool
	fromConfig bool
	update     bool
}

func (a *App) createPayloads(o *getOutputResult, status outputsStatus) []displayPayload {
	var enableLaptop, enableExternals bool
	switch status {
	case statusWELO:
		enableLaptop = true
		enableExternals = true
	case statusOLLO, statusOLLC:
		enableLaptop = true
	case statusWELC:
		enableExternals = true
	}

	slog.Debug("status processed", "enable_laptop", enableLaptop, "enable_externals", enableExternals)
	var payloads []displayPayload

	// specific checks for if laptop display needs to be enabled or disabled
	var lp *displayPayload
	if enableLaptop && !a.laptopDisplayEnabled(o) {
		lp = &displayPayload{
			in:         a.Cfg.LaptopDisplay,
			out:        a.Cfg.LaptopDisplay,
			enable:     true,
			fromConfig: true,
			update:     true,
		}
	} else if !enableLaptop && a.laptopDisplayEnabled(o) {
		lp = &displayPayload{
			in:         a.Cfg.LaptopDisplay,
			out:        a.Cfg.LaptopDisplay,
			enable:     false,
			fromConfig: true,
			update:     true,
		}
	}

	if lp != nil {
		payloads = append(payloads, *lp)
	}

	for _, m := range o.displays {
		if p := a.createPayload(m, enableLaptop, enableExternals); p != nil {
			payloads = append(payloads, *p)
		}
	}

	return payloads
}

func (a *App) laptopDisplayEnabled(o *getOutputResult) bool {
	return displayEnabled(o, a.Cfg.LaptopDisplay)
}

func (a *App) createPayload(in hypr.Monitor, enableLaptop, enableExternals bool) *displayPayload {
	p := &displayPayload{
		in: in,
	}

	p.out = in
	if c, ok := a.getDisplayFromConfig(in); ok {
		p.fromConfig = true
		p.out = c
	}

	p.enable = enableExternals
	if a.isLaptopDisplay(in) {
		p.enable = enableLaptop
	}

	if displayUpdateNeeded(p.in, p.out) {
		p.update = true
	}

	return p
}

func displayUpdateNeeded(a, b hypr.Monitor) bool {
	return !reflect.DeepEqual(a, b)
}

func displayEnabled(o *getOutputResult, m hypr.Monitor) bool {
	_, ok := o.displays[m.Name]
	return ok
}
