package app

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/dsrosen6/hyprlaptop/internal/hypr"
	"github.com/dsrosen6/hyprlaptop/internal/lid"
)

type (
	getOutputResult struct {
		laptopName string
		monitors   hypr.MonitorMap
		lidState   lid.State
	}

	outputsStatus string
)

const (
	statusUnknown outputsStatus = "UNKNOWN"
	statusOLLC    outputsStatus = "ONLY_LAPTOP_LID_CLOSED"
	statusOLLO    outputsStatus = "ONLY_LAPTOP_LID_OPEN"
	statusWELC    outputsStatus = "WITH_EXTERNAL_LID_CLOSED"
	statusWELO    outputsStatus = "WITH_EXTERNAL_LID_OPEN"
)

func (a *App) Run() error {
	o, err := a.getOutputs()
	if err != nil {
		return fmt.Errorf("getting output info: %w", err)
	}

	s := o.statusShouldBe()
	slog.Info(fmt.Sprintf("status should be: %s", s))

	if err := a.setOutputs(s, o); err != nil {
		return fmt.Errorf("setting outputs: %w", err)
	}
	return nil
}

func (a *App) setOutputs(status outputsStatus, o *getOutputResult) error {
	slog.Debug("got status", "status", status)
	if slices.Contains([]outputsStatus{statusOLLC, statusOLLO}, status) {
		return a.setOutputsOnlyLaptop(status)
	} else if slices.Contains([]outputsStatus{statusWELC, statusWELO}, status) {
		return a.setOutputsWithExternal(status, o)
	}

	return fmt.Errorf("got unknown status: %s", status)
}

func (a *App) setOutputsOnlyLaptop(status outputsStatus) error {
	slog.Debug("running: setOutputOnlyLaptop", "status", status)
	switch status {
	case statusOLLC:
		// TODO: implement
		slog.Info("statusOLLC setter needs to be implemented")
		return nil
	case statusOLLO:
		slog.Info("statusOLLO setter needs to be implemented")
		return nil
	default:
		return fmt.Errorf("unknown status: %s", status)
	}
}

func (a *App) setOutputsWithExternal(status outputsStatus, o *getOutputResult) error {
	slog.Debug("running: setOutputsWithExternal", "status", status)
	var err error
	switch status {
	case statusWELC:
		err = a.setClamshellMode()
	case statusWELO:
		err = a.setExternalLidOpen(o)
	default:
		err = fmt.Errorf("unknown status: %s", status)
	}

	return err
}

func (a *App) setClamshellMode() error {
	slog.Debug("running: setClamshellMode")
	if err := a.Hctl.DisableMonitor(a.Cfg.LaptopMonitor); err != nil {
		return fmt.Errorf("disabling laptop monitor: %w", err)
	}
	slog.Info("clamshell mode set; disabled laptop display", "display_name", a.Cfg.LaptopMonitor.Name)
	return nil
}

func (a *App) setExternalLidOpen(o *getOutputResult) error {
	slog.Debug("running: setExternalLidOpen")
	wg := new(sync.WaitGroup)
	errc := make(chan error, len(o.monitors))

	// compare laptop monitors to see if updated is needed
	// once checked, delete it from the map to exclude it from external display processing
	fromConfig := false
	got := o.monitors[o.laptopName]
	lm := got
	if a.Cfg.LaptopMonitor != (hypr.Monitor{}) {
		fromConfig = true
		lm = a.Cfg.LaptopMonitor
	}
	slog.Debug("got laptop monitor", logMonitorAttr(lm, fromConfig))

	delete(o.monitors, lm.Name)
	slog.Debug("deleted from map: laptop display", "name", a.Cfg.LaptopMonitor.Name)

	if displayUpdateNeeded(got, lm) {
		slog.Info("changes needed on laptop display", "name", lm.Name)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := a.Hctl.EnableMonitor(lm); err != nil {
				errc <- fmt.Errorf("enabling laptop monitor with name %s: %w", lm.Name, err)
			}
			slog.Info("laptop display enabled or updated", "name", lm.Name)
		}()
	} else {
		slog.Info("no changes needed on laptop display", "name", lm.Name)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.enableExternalMonitors(o.monitors); err != nil {
			errc <- fmt.Errorf("enabling external monitors: %w", err)
		}
	}()

	go func() {
		wg.Wait()
		close(errc)
	}()

	var hasErr bool
	for err := range errc {
		hasErr = true
		slog.Error(err.Error())
	}

	if hasErr {
		return errors.New("one or more monitors failed to enable; see logs")
	}

	return nil
}

func (a *App) enableExternalMonitors(monitors hypr.MonitorMap) error {
	wg := new(sync.WaitGroup)
	errc := make(chan error, len(monitors))

	for _, in := range monitors {
		wg.Add(1)
		go func(in hypr.Monitor) {
			defer wg.Done()
			m := in
			if cm, ok := a.Cfg.ExternalMonitors[in.Name]; ok {
				m = cm
			}

			if displayUpdateNeeded(in, m) {
				if err := a.Hctl.EnableMonitor(m); err != nil {
					errc <- fmt.Errorf("setting external monitor %s: %w", m.Name, err)
				}
				slog.Info("external display enabled", "name", m.Name)
			} else {
				slog.Info("no changes needed on external display", "name", m.Name)
			}
		}(in)
	}

	go func() {
		wg.Wait()
		close(errc)
	}()

	hasErr := false
	for err := range errc {
		hasErr = true
		slog.Error(err.Error())
	}

	if hasErr {
		return errors.New("failed to enable one or more external monitors; see logs")
	}

	return nil
}

func (a *App) getOutputs() (*getOutputResult, error) {
	current, err := a.Hctl.ListMonitors()
	if err != nil {
		return nil, fmt.Errorf("listing current monitors: %w", err)
	}

	var names []string
	for _, m := range current {
		names = append(names, m.Name)
	}
	slog.Info("monitors detected", "names", strings.Join(names, ","))

	ls, err := lid.GetState()
	if err != nil {
		return nil, fmt.Errorf("getting lid status: %w", err)
	}
	slog.Info(fmt.Sprintf("lid state: %s", ls))

	return &getOutputResult{
		laptopName: a.Cfg.LaptopMonitor.Name,
		monitors:   current,
		lidState:   ls,
	}, nil
}

// statusShouldBe checks the state of monitors and lid status, and returns the status
// that hyprlaptop should be switched to (if it isn't already)
func (o *getOutputResult) statusShouldBe() outputsStatus {
	// check if laptop is the only monitor
	if _, ok := o.monitors[o.laptopName]; ok && len(o.monitors) == 1 {
		return onlyLaptopStates(o.lidState)
	}

	return withExternalStates(o.lidState)
}

func onlyLaptopStates(ls lid.State) outputsStatus {
	switch ls {
	case lid.StateOpen:
		return statusOLLO
	case lid.StateClosed:
		return statusOLLC
	case lid.StateUnknown:
		return statusUnknown
	default:
		return statusUnknown
	}
}

func withExternalStates(ls lid.State) outputsStatus {
	switch ls {
	case lid.StateOpen:
		return statusWELO
	case lid.StateClosed:
		return statusWELC
	case lid.StateUnknown:
		return statusUnknown
	default:
		return statusUnknown
	}
}

func logMonitorAttr(m hypr.Monitor, fromConfig bool) slog.Attr {
	return slog.Group(
		m.Name,
		slog.Bool("from_config", fromConfig),
		slog.Int64("width", m.Width),
		slog.Int64("height", m.Height),
		slog.Float64("refresh_rate", m.RefreshRate),
		slog.Int64("x", m.X),
		slog.Int64("y", m.Y),
		slog.Float64("scale", m.Scale),
	)
}

func displayUpdateNeeded(a, b hypr.Monitor) bool {
	return !reflect.DeepEqual(a, b)
}
