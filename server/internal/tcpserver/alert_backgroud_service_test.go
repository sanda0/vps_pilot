package tcpserver

import "testing"

func TestNetOverThreshold(t *testing.T) {
	cases := []struct {
		name                 string
		sent, recv           float64
		sendLimit, recvLimit float64
		want                 bool
	}{
		{"both under", 100, 100, 1000, 1000, false},
		{"sent over", 1500, 100, 1000, 1000, true},
		{"recv over", 100, 1500, 1000, 1000, true},
		{"equal is not over", 1000, 1000, 1000, 1000, false},
		{"sent checked against send limit only", 1500, 100, 2000, 1000, false},
		{"recv checked against recv limit only", 100, 1500, 1000, 2000, false},
		{"empty send limit not checked", 1e9, 100, 0, 1000, false},
		{"empty recv limit not checked", 100, 1e9, 1000, 0, false},
		{"both empty never fires", 1e9, 1e9, 0, 0, false},
	}
	for _, c := range cases {
		if got := netOverThreshold(c.sent, c.recv, c.sendLimit, c.recvLimit); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestNetThresholdText(t *testing.T) {
	if got := netThresholdText(0); got != "off" {
		t.Errorf("netThresholdText(0) = %q, want off", got)
	}
	if got := netThresholdText(2500); got != "2.5 KB/s" {
		t.Errorf("netThresholdText(2500) = %q, want 2.5 KB/s", got)
	}
}

func TestFormatBytesPerSec(t *testing.T) {
	cases := map[float64]string{
		0:       "0 B/s",
		999:     "999 B/s",
		1000:    "1.0 KB/s",
		1500000: "1.5 MB/s",
		2e9:     "2.0 GB/s",
		3e12:    "3000.0 GB/s",
	}
	for in, want := range cases {
		if got := formatBytesPerSec(in); got != want {
			t.Errorf("formatBytesPerSec(%v) = %q, want %q", in, got, want)
		}
	}
}
