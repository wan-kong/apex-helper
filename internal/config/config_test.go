package config

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "nested", "config.json")}
	c, err := s.Load()
	if err != nil || !c.Profiles.Optimize.WinKeyDisabled {
		t.Fatalf("defaults: %#v, %v", c, err)
	}
	c.Paths.Steam = `C:\Steam\steam.exe`
	c.Profiles.Restore.AudioDeviceID = "speakers"
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	c.Shortcut = "Ctrl+Alt+F9"
	if err := s.Save(c); err != nil {
		t.Fatalf("replace existing config: %v", err)
	}
	got, err := s.Load()
	if err != nil || got.Paths.Steam != c.Paths.Steam || got.Profiles.Restore.AudioDeviceID != "speakers" || got.Shortcut != c.Shortcut {
		t.Fatalf("round trip: %#v, %v", got, err)
	}
}
