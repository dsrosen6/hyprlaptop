package hyprctl

func (c *Client) ListMonitors() ([]Monitor, error) {
	var m []Monitor
	if err := c.RunCommandWithUnmarshal([]string{"monitors"}, &m); err != nil {
		return nil, err
	}

	return m, nil
}
