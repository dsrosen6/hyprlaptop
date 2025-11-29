package hyprctl

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	_, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_RunCommand(t *testing.T) {
	c, err := newTestClient(t)
	if err != nil {
		t.Fatalf("error creating test client: %v", err)
	}

	_, err = c.RunCommand([]string{"-j", "monitors"})
	if err != nil {
		t.Fatalf("error running monitors command: %v", err)
	}
}

func TestClient_RunCommand_WantErr(t *testing.T) {
	c, err := newTestClient(t)
	if err != nil {
		t.Fatalf("error creating test client: %v", err)
	}

	out, err := c.RunCommand([]string{"-j", "monitrs"})
	if err == nil {
		t.Fatalf("wanted error, did not get error; out: %s", out)
	}
}

func TestClient_ListMonitors(t *testing.T) {
	c, err := newTestClient(t)
	if err != nil {
		t.Fatalf("error creating test client: %v", err)
	}

	monitors, err := c.ListMonitors()
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range monitors {
		t.Logf("Monitor found: %s", m.Name)
	}
}

func newTestClient(t *testing.T) (*Client, error) {
	t.Helper()
	c, err := NewClient()
	if err != nil {
		return nil, err
	}

	return c, nil
}
