package listener

type Event struct {
	Type    EventType
	Details string
}

// EventType is mostly for logging, but this may change in the future.
type EventType string

const (
	ConfigUpdatedEvent  EventType = "CONFIG_UPDATED"
	DisplayAddEvent     EventType = "DISPLAY_ADDED"
	DisplayRemoveEvent  EventType = "DISPLAY_REMOVED"
	DisplayUnknownEvent EventType = "DISLAY_UNKNOWN_EVENT"
	IdleWakeEvent       EventType = "IDLE_WAKE"
	LidSwitchEvent      EventType = "LID_SWITCH"
)
