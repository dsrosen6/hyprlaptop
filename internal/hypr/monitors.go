package hypr

import (
	"fmt"
)

type (
	Monitor struct {
		Name        string  `json:"name,omitempty"`
		Description string  `json:"description,omitempty"`
		Make        string  `json:"make,omitempty"`
		Model       string  `json:"model,omitempty"`
		Width       int64   `json:"width,omitempty"`
		Height      int64   `json:"height,omitempty"`
		RefreshRate float64 `json:"refreshRate,omitempty"`
		X           int64   `json:"x,omitempty"`
		Y           int64   `json:"y,omitempty"`
		Scale       float64 `json:"scale,omitempty"`
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

func (h *HyprctlClient) EnableOrUpdateMonitor(m Monitor) error {
	args := []string{"keyword", "monitor", monitorToConfigString(m)}
	if _, err := h.RunCommand(args); err != nil {
		return err
	}

	return nil
}

func (h *HyprctlClient) DisableMonitor(m Monitor) error {
	args := []string{"keyword", "monitor", m.Name + ",", "disable"}
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
