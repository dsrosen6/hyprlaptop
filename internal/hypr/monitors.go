package hypr

import (
	"fmt"
)

type (
	Monitor struct {
		Name        string  `json:"name"`
		Width       int64   `json:"width"`
		Height      int64   `json:"height"`
		RefreshRate float64 `json:"refreshRate"`
		X           int64   `json:"x"`
		Y           int64   `json:"y"`
		Scale       float64 `json:"scale"`
	}

	MonitorMap map[string]Monitor
)

func (h *HyprctlClient) ListMonitors() (MonitorMap, error) {
	var monitors []Monitor
	if err := h.RunCommandWithUnmarshal([]string{"monitors"}, &monitors); err != nil {
		return nil, err
	}

	mm := make(MonitorMap, len(monitors))
	for _, m := range monitors {
		mm[m.Name] = m
	}

	return mm, nil
}

func (h *HyprctlClient) EnableMonitor(m Monitor) error {
	args := []string{"keyword", "monitor", monitorToConfigString(m)}
	if _, err := h.RunCommand(args); err != nil {
		return err
	}

	return nil
}

func (h *HyprctlClient) DisableMonitor(m Monitor) error {
	args := []string{"keyword", "monitor", m.Name, "disable"}
	if _, err := h.RunCommand(args); err != nil {
		return err
	}

	return nil
}

func monitorToConfigString(m Monitor) string {
	res := fmt.Sprintf("%dx%d", m.Width, m.Height)
	res = fmt.Sprintf("%s@%f", res, m.RefreshRate)
	xy := fmt.Sprintf("%dx%d", m.X, m.Y)
	scale := fmt.Sprintf("%f", m.Scale)
	return fmt.Sprintf("%s,%s,%s,%s", m.Name, res, xy, scale)
}
