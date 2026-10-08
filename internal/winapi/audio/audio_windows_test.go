package audio

import "testing"

func TestEnumerateDevices(t *testing.T) {
	devices, defaultID, err := Enumerate()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range devices {
		if d.ID == "" || d.Name == "" {
			t.Fatalf("incomplete device: %#v", d)
		}
	}
	t.Logf("active output devices: %d, default ID present: %t", len(devices), defaultID != "")
}
