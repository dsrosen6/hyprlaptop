package main

func (c *client) listMonitors() (map[string]Monitor, error) {
	var monitors []Monitor
	if err := c.runCommandWithUnmarshal([]string{"monitors"}, &monitors); err != nil {
		return nil, err
	}

	mm := make(map[string]Monitor, len(monitors))
	for _, m := range monitors {
		mm[m.Name] = m
	}

	return mm, nil
}
