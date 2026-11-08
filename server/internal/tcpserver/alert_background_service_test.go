package tcpserver

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sanda0/vps_pilot/internal/db"
)

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

// A disk alert notifies only above its threshold, labelled "Disk".
func TestCheckPercentUsageDisk(t *testing.T) {
	h := newTestHub(t)
	ctx := context.Background()
	lastAlertSentTime = make(map[int32]time.Time)

	var got []string
	discord := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, string(body))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer discord.Close()

	if _, err := h.repo.OperationalDB.Exec(`INSERT INTO nodes (id, name, ip, agent_id) VALUES (1, 'web', '10.0.0.1', 'a')`); err != nil {
		t.Fatal(err)
	}
	if _, err := h.repo.Queries.CreateAlert(ctx, db.CreateAlertParams{
		NodeID:         1,
		Metric:         "disk",
		Threshold:      sql.NullFloat64{Float64: 70, Valid: true},
		DiscordWebhook: sql.NullString{String: discord.URL, Valid: true},
		IsActive:       sql.NullInt64{Int64: 1, Valid: true},
	}); err != nil {
		t.Fatal(err)
	}

	checkPercentUsage(ctx, h.repo, 1, "disk", "Disk", 60)
	if len(got) != 0 {
		t.Fatalf("60%% under a 70%% threshold sent %d notifications", len(got))
	}

	checkPercentUsage(ctx, h.repo, 1, "disk", "Disk", 80)
	if len(got) != 1 {
		t.Fatalf("80%% over a 70%% threshold sent %d notifications, want 1", len(got))
	}
	for _, want := range []string{"**Disk**", "80.00%", "70.00%"} {
		if !strings.Contains(got[0], want) {
			t.Errorf("notification %s is missing %q", got[0], want)
		}
	}
}
