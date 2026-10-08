package keyboard

import "testing"

func TestHookLifecycle(t *testing.T) {
	s := New()
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	s.SetWinDisabled(true)
	s.SetAltShiftDisabled(true)
	if !s.WinDisabled() || !s.AltShiftDisabled() {
		t.Fatal("flags were not applied")
	}
	s.Stop()
}
