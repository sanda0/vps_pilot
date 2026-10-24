package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

// Every time range button in the dashboard must reach the query as the right
// number of seconds.
func TestTimeRangeButtonsToSeconds(t *testing.T) {
	for button, want := range map[string]int64{
		"5M":  5 * 60,
		"15M": 15 * 60,
		"1H":  3600,
		"1D":  86400,
		"2D":  2 * 86400,
		"7D":  7 * 86400,
		"1W":  7 * 86400,
	} {
		var req dto.NodeSystemStatRequestDto
		if err := req.FromBytes([]byte(fmt.Sprintf(`{"id":1,"time_range":%q}`, button))); err != nil {
			t.Fatal(err)
		}
		if got := parseTimeRangeToSeconds(req.TimeRange); got != want {
			t.Errorf("%s: got %d seconds, want %d", button, got, want)
		}
	}
}

type fakeAgents map[int64]bool

func (f fakeAgents) Kick(nodeID int64) bool   { return f[nodeID] }
func (f fakeAgents) Online(nodeID int64) bool { return f[nodeID] }

func TestNodeStatus(t *testing.T) {
	op, ts, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { op.Close(); ts.Close() })
	repo := db.NewRepo(op, ts)
	if _, err := ts.Exec(`INSERT INTO net_stat (timestamp, node_id, sent, recv) VALUES (100, 1, 0, 0), (200, 1, 0, 0)`); err != nil {
		t.Fatal(err)
	}

	svc := NewNodeService(context.Background(), repo, AgentEnrollment{Agents: fakeAgents{1: true}})

	if got := svc.Status(1); !got.Online || got.LastSeen != 200 {
		t.Errorf("connected node: got %+v, want online, last seen 200", got)
	}
	if got := svc.Status(2); got.Online || got.LastSeen != 0 {
		t.Errorf("never seen node: got %+v, want offline, last seen 0", got)
	}
}

func TestInstallCommand(t *testing.T) {
	base := "curl -fsSL https://vpspilot.dev/install-agent.sh | sudo VPS_PILOT_TOKEN=vpsp1.x sh -s -- --hub hub.example.com:55001"
	if got := installCommand("vpsp1.x", "hub.example.com:55001", "v0.2.0"); got != base+" --version v0.2.0" {
		t.Errorf("release hub: got %q", got)
	}
	if got := installCommand("vpsp1.x", "hub.example.com:55001", "dev"); got != base {
		t.Errorf("dev hub: got %q", got)
	}
}
