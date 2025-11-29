package main

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

func pickMonitor(c *client) (*Monitor, error) {
	monitors, err := c.listMonitors()
	if err != nil {
		return nil, fmt.Errorf("listing monitors: %w", err)
	}

	var opts []string
	for _, m := range monitors {
		opts = append(opts, m.Name)
	}

	var sel string
	grp := huh.NewGroup(
		huh.NewSelect[string]().
			Title("Select your laptop display").
			Options(huh.NewOptions(opts...)...).
			Value(&sel),
	)

	f := form(grp)
	if err := f.Run(); err != nil {
		return nil, fmt.Errorf("running monitor selection form: %w", err)
	}

	m := monitors[sel]
	return &m, nil
}

func form(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).
		WithTheme(huh.ThemeBase())
}
