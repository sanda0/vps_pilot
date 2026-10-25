package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

type fakeRunner struct {
	repo    *db.Repo
	err     error
	started []tcpserver.StartRun
	events  chan tcpserver.RunEvent
}

func (f *fakeRunner) Start(ctx context.Context, req tcpserver.StartRun) (db.CommandRun, error) {
	if f.err != nil {
		return db.CommandRun{}, f.err
	}
	f.started = append(f.started, req)
	return f.repo.Queries.CreateCommandRun(ctx, db.CreateCommandRunParams{
		ID: "run-1", NodeID: req.NodeID, ProjectPath: req.ProjectPath, ProjectName: req.ProjectName,
		Command: req.Command, CommandLine: req.CommandLine, Username: req.Username,
		UserID: sql.NullInt64{Int64: req.UserID, Valid: true},
	})
}
func (f *fakeRunner) Cancel(string) error { return tcpserver.ErrRunNotRunning }
func (f *fakeRunner) Watch(string) (<-chan tcpserver.RunEvent, func()) {
	return f.events, func() {}
}

func newRunTestServer(t *testing.T) (*gin.Engine, *fakeRunner) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	op, ts, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { op.Close(); ts.Close() })
	for _, q := range []string{
		`INSERT INTO users (id, username, email, password_hash) VALUES (1, 'admin', 'a@example.com', 'x')`,
		`INSERT INTO nodes (id, name) VALUES (1, 'web')`,
		`INSERT INTO projects (id, node_id, name, path, commands)
		 VALUES ('p1', 1, 'shop', '/var/www/shop', '[{"name":"build","command":"npm run build"}]')`,
	} {
		if _, err := op.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	repo := db.NewRepo(op, ts)
	runner := &fakeRunner{repo: repo, events: make(chan tcpserver.RunEvent, 8)}
	h := NewCommandRunHandler(services.NewCommandRunService(context.Background(), repo, runner))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int32(1)) })
	r.POST("/projects/:id/runs", h.Start)
	r.GET("/projects/:id/runs", h.List)
	r.GET("/runs/:runId", h.Get)
	r.POST("/runs/:runId/cancel", h.Cancel)
	r.GET("/runs/:runId/ws", h.Watch)
	return r, runner
}

func do(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w
}

func TestCommandRunAPI(t *testing.T) {
	r, runner := newRunTestServer(t)

	for _, tc := range []struct {
		path, body string
		want       int
	}{
		{"/projects/nope/runs", `{"command":"build"}`, http.StatusNotFound},
		{"/projects/p1/runs", `{}`, http.StatusBadRequest},
		{"/projects/p1/runs", `{"command":"rm -rf /"}`, http.StatusBadRequest},
	} {
		if w := do(r, "POST", tc.path, tc.body); w.Code != tc.want {
			t.Errorf("POST %s %s: %d %s, want %d", tc.path, tc.body, w.Code, w.Body, tc.want)
		}
	}
	if len(runner.started) != 0 {
		t.Fatalf("started %+v for invalid requests", runner.started)
	}

	runner.err = tcpserver.ErrNoCommands
	if w := do(r, "POST", "/projects/p1/runs", `{"command":"build"}`); w.Code != http.StatusConflict {
		t.Fatalf("v0.2 agent: %d %s", w.Code, w.Body)
	}
	runner.err = nil
	w := do(r, "POST", "/projects/p1/runs", `{"command":"build"}`)
	want := tcpserver.StartRun{NodeID: 1, ProjectPath: "/var/www/shop", ProjectName: "shop",
		Command: "build", CommandLine: "npm run build", UserID: 1, Username: "admin"}
	if w.Code != http.StatusAccepted || len(runner.started) != 1 || runner.started[0] != want {
		t.Fatalf("start: %d %s, started %+v", w.Code, w.Body, runner.started)
	}

	if w := do(r, "GET", "/projects/p1/runs", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"id":"run-1"`) {
		t.Fatalf("list: %d %s", w.Code, w.Body)
	}
	if w := do(r, "GET", "/runs/run-1", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"output":[]`) {
		t.Fatalf("get: %d %s", w.Code, w.Body)
	}
	if w := do(r, "POST", "/runs/run-1/cancel", ""); w.Code != http.StatusConflict {
		t.Fatalf("cancel: %d %s", w.Code, w.Body)
	}
	if w := do(r, "GET", "/runs/nope", ""); w.Code != http.StatusNotFound {
		t.Fatalf("missing run: %d", w.Code)
	}
}

func TestCommandRunWebSocket(t *testing.T) {
	r, runner := newRunTestServer(t)
	do(r, "POST", "/projects/p1/runs", `{"command":"build"}`)
	ctx := context.Background()
	runner.repo.Queries.AddCommandRunOutput(ctx, db.AddCommandRunOutputParams{RunID: "run-1", Seq: 0, Stream: "stdout", Data: "a"})

	srv := httptest.NewServer(r)
	defer srv.Close()
	url := "ws" + srv.URL[len("http"):] + "/runs/run-1/ws"
	if _, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"https://evil.example"}}); err == nil {
		t.Fatal("cross-origin WebSocket accepted")
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var got []string
	for {
		var msg struct {
			Run    *struct{ Status string }
			Output *struct{ Data string }
		}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		if msg.Run != nil {
			got = append(got, "run:"+msg.Run.Status)
		} else {
			got = append(got, "output:"+msg.Output.Data)
		}
		if len(got) == 2 {
			// Live events: seq 0 again (already sent, skipped), seq 1, then the exit.
			runner.events <- tcpserver.RunEvent{Output: &tcpserver.CommandOutput{Seq: 0, Stream: "stdout", Data: "a"}}
			runner.events <- tcpserver.RunEvent{Output: &tcpserver.CommandOutput{Seq: 1, Stream: "stderr", Data: "b"}}
			runner.repo.Queries.FinishCommandRun(ctx, db.FinishCommandRunParams{Status: "exited", ID: "run-1"})
			runner.events <- tcpserver.RunEvent{Exit: &tcpserver.CommandExit{Status: "exited"}}
		}
	}
	if b, _ := json.Marshal(got); string(b) != `["run:running","output:a","output:b","run:exited"]` {
		t.Fatalf("messages %s", b)
	}
}
