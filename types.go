package main

// Monitor matches the output of 'hyprctl monitors', and is also used for config.
type Monitor struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Make             string    `json:"make"`
	Model            string    `json:"model"`
	Serial           string    `json:"serial"`
	Width            int64     `json:"width"`
	Height           int64     `json:"height"`
	RefreshRate      float64   `json:"refreshRate"`
	X                int64     `json:"x"`
	Y                int64     `json:"y"`
	ActiveWorkspace  Workspace `json:"activeWorkspace"`
	SpecialWorkspace Workspace `json:"specialWorkspace"`
	Reserved         []int64   `json:"reserved"`
	Scale            float64   `json:"scale"`
	Transform        int64     `json:"transform"`
	Focused          bool      `json:"focused"`
	DPMSStatus       bool      `json:"dpmsStatus"`
	Vrr              bool      `json:"vrr"`
	Solitary         string    `json:"solitary"`
	ActivelyTearing  bool      `json:"activelyTearing"`
	DirectScanoutTo  string    `json:"directScanoutTo"`
	Disabled         bool      `json:"disabled"`
	CurrentFormat    string    `json:"currentFormat"`
	MirrorOf         string    `json:"mirrorOf"`
	AvailableModes   []string  `json:"availableModes"`
}

type Workspace struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
