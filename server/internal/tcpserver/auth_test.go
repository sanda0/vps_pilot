package tcpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/sanda0/vps_pilot/internal/agentauth"
	"github.com/sanda0/vps_pilot/internal/db"
)

type testHub struct {
	t        *testing.T
	repo     *db.Repo
	router   *Router
	registry *Registry
	runs     *Runs
	logs     *Logs
	backups  *Backups
	chans    statChans
}

func newTestHub(t *testing.T) *testHub {
	t.Helper()
	op, ts, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { op.Close(); ts.Close() })

	h := &testHub{
		t:        t,
		repo:     db.NewRepo(op, ts),
		router:   NewRouter(),
		registry: NewRegistry(),
		chans:    statChans{stat: make(chan Msg, 10), monitor: make(chan Msg, 10)},
	}
	h.runs = NewRuns(context.Background(), h.repo, h.registry)
	h.logs = NewLogs(h.registry)
	h.backups = NewBackups(context.Background(), h.repo)
	registerHandlers(h.router, h.chans, h.runs, h.logs, h.backups)
	return h
}

// joinSecret stores a join token and returns its secret part.
func (h *testHub) joinSecret(nodeID *int64, ttl time.Duration) string {
	h.t.Helper()
	token, hash, err := agentauth.NewJoinToken(agentauth.Fingerprint([]byte("cert")))
	if err != nil {
		h.t.Fatal(err)
	}
	ref := sql.NullInt64{}
	if nodeID != nil {
		ref = sql.NullInt64{Int64: *nodeID, Valid: true}
	}
	if _, err := h.repo.Queries.CreateJoinToken(context.Background(), db.CreateJoinTokenParams{
		TokenHash: hash,
		NodeID:    ref,
		CreatedBy: 1,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}); err != nil {
		h.t.Fatal(err)
	}
	secret, _, err := agentauth.ParseJoinToken(token)
	if err != nil {
		h.t.Fatal(err)
	}
	return secret
}

// dial opens an in-memory agent connection served by the hub's router.
func (h *testHub) dial() *testAgent {
	h.t.Helper()
	server, client := net.Pipe()
	h.t.Cleanup(func() { client.Close() })
	c := &ConnContext{RemoteAddr: "10.0.0.5:4000", conn: server, repo: h.repo}
	go h.router.serveConn(context.Background(), c, h.registry)
	client.SetDeadline(time.Now().Add(5 * time.Second))
	return &testAgent{t: h.t, conn: client}
}

type testAgent struct {
	t    *testing.T
	conn net.Conn
}

func (a *testAgent) send(msgType, token string, data any) {
	a.t.Helper()
	raw, _ := json.Marshal(data)
	if err := WriteFrame(a.conn, Msg{Msg: msgType, Token: token, Data: raw}); err != nil {
		a.t.Fatal(err)
	}
}

func (a *testAgent) expect(msgType string) Msg {
	a.t.Helper()
	var msg Msg
	if err := ReadFrame(a.conn, &msg); err != nil {
		a.t.Fatalf("waiting for %q: %v", msgType, err)
	}
	if msg.Msg != msgType {
		a.t.Fatalf("got %q (%s), want %q", msg.Msg, msg.Data, msgType)
	}
	return msg
}

func (a *testAgent) enroll(secret string) EnrolledPayload {
	a.t.Helper()
	a.send(MsgEnroll, secret, SystemInfo{Hostname: "web-1", OS: "linux", CPUs: 2})
	var creds EnrolledPayload
	if err := json.Unmarshal(a.expect(MsgEnrolled).Data, &creds); err != nil {
		a.t.Fatal(err)
	}
	a.expect(MsgSysStat)
	a.expect(MsgProjectSettings)
	return creds
}

func connectToken(c EnrolledPayload) string { return c.AgentID + ":" + c.NodeKey }

func TestEnrollThenConnect(t *testing.T) {
	h := newTestHub(t)
	creds := h.dial().enroll(h.joinSecret(nil, time.Hour))

	node, err := h.repo.Queries.GetNode(context.Background(), creds.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	if node.Name.String != "web-1" || node.Ip != "10.0.0.5" {
		t.Errorf("unexpected node %+v", node)
	}
	if node.KeyHash.String == creds.NodeKey {
		t.Error("node key stored in plain text")
	}

	agent := h.dial()
	agent.send(MsgConnected, connectToken(creds), SystemInfo{OS: "linux"})
	if got := agent.expect(MsgSysStat).NodeId; int64(got) != creds.NodeID {
		t.Errorf("sys_stat node id %d, want %d", got, creds.NodeID)
	}
	agent.expect(MsgProjectSettings)

	// Stats now flow, stamped with the authenticated node ID.
	agent.send(MsgSysStat, "", SystemStat{MemUsage: 10})
	select {
	case msg := <-h.chans.stat:
		if int64(msg.NodeId) != creds.NodeID {
			t.Errorf("stat for node %d, want %d", msg.NodeId, creds.NodeID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stat not forwarded")
	}
	if _, ok := h.registry.Get(creds.NodeID); !ok {
		t.Error("connection not registered")
	}
}

func TestJoinTokenIsSingleUse(t *testing.T) {
	h := newTestHub(t)
	secret := h.joinSecret(nil, time.Hour)
	h.dial().enroll(secret)

	agent := h.dial()
	agent.send(MsgEnroll, secret, SystemInfo{})
	agent.expect(MsgAuthFailed)
}

func TestExpiredJoinTokenRejected(t *testing.T) {
	h := newTestHub(t)
	agent := h.dial()
	agent.send(MsgEnroll, h.joinSecret(nil, -time.Minute), SystemInfo{})
	agent.expect(MsgAuthFailed)
}

func TestConnectRejectsBadCredentials(t *testing.T) {
	h := newTestHub(t)
	creds := h.dial().enroll(h.joinSecret(nil, time.Hour))

	for name, token := range map[string]string{
		"wrong key":     creds.AgentID + ":nope",
		"unknown agent": "nope:" + creds.NodeKey,
		"no token":      "",
	} {
		t.Run(name, func(t *testing.T) {
			agent := h.dial()
			agent.send(MsgConnected, token, SystemInfo{})
			agent.expect(MsgAuthFailed)
		})
	}
}

func TestRevokedKeyRejected(t *testing.T) {
	h := newTestHub(t)
	creds := h.dial().enroll(h.joinSecret(nil, time.Hour))
	if _, err := h.repo.Queries.ClearNodeKey(context.Background(), creds.NodeID); err != nil {
		t.Fatal(err)
	}
	agent := h.dial()
	agent.send(MsgConnected, connectToken(creds), SystemInfo{})
	agent.expect(MsgAuthFailed)
}

func TestReenrollKeepsNodeAndRotatesKey(t *testing.T) {
	h := newTestHub(t)
	old := h.dial().enroll(h.joinSecret(nil, time.Hour))

	fresh := h.dial().enroll(h.joinSecret(&old.NodeID, time.Hour))
	if fresh.NodeID != old.NodeID {
		t.Fatalf("re-enroll created node %d, want %d", fresh.NodeID, old.NodeID)
	}

	agent := h.dial()
	agent.send(MsgConnected, connectToken(old), SystemInfo{})
	agent.expect(MsgAuthFailed)

	agent = h.dial()
	agent.send(MsgConnected, connectToken(fresh), SystemInfo{})
	agent.expect(MsgSysStat)
}

func TestUnauthenticatedMessagesDropped(t *testing.T) {
	h := newTestHub(t)
	agent := h.dial()
	agent.send(MsgSysStat, "", SystemStat{MemUsage: 10})
	agent.send(MsgProjects, "", ProjectSyncPayload{})

	select {
	case <-h.chans.stat:
		t.Fatal("stat from unauthenticated agent was forwarded")
	case <-time.After(200 * time.Millisecond):
	}
}

func TestHandshakeTimeoutClosesConnection(t *testing.T) {
	old := HandshakeTimeout
	HandshakeTimeout = 100 * time.Millisecond
	t.Cleanup(func() { HandshakeTimeout = old })

	h := newTestHub(t)
	agent := h.dial()
	if _, err := agent.conn.Read(make([]byte, 1)); err == nil {
		t.Fatal("expected idle unauthenticated connection to be closed")
	}
}

// conn waits for the node's registered connection.
func (h *testHub) conn(nodeID int64) *ConnContext {
	h.t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if c, ok := h.registry.Get(nodeID); ok {
			return c
		}
	}
	h.t.Fatalf("node %d never registered", nodeID)
	return nil
}

func TestConnectRecordsFeatures(t *testing.T) {
	h := newTestHub(t)
	creds := h.dial().enroll(h.joinSecret(nil, time.Hour))
	if h.conn(creds.NodeID).Supports(FeatureCommands) {
		t.Fatal("agent without features supports commands")
	}

	agent := h.dial()
	agent.send(MsgConnected, connectToken(creds), SystemInfo{AgentVersion: "v0.3.0", Features: []string{FeatureCommands}})
	agent.expect(MsgSysStat)
	agent.expect(MsgProjectSettings)
	for deadline := time.Now().Add(2 * time.Second); !h.conn(creds.NodeID).Supports(FeatureCommands); time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("features not recorded on the new connection")
		}
	}
}
