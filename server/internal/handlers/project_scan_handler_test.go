package handlers

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/services"
)

type fakeAgents struct {
	online bool
	pushed []int64
}

func (f *fakeAgents) PushProjectSettings(_ context.Context, nodeID int64) {
	f.pushed = append(f.pushed, nodeID)
}
func (f *fakeAgents) RequestProjectRescan(int64) (bool, error) { return f.online, nil }
func (f *fakeAgents) Online(int64) bool                        { return f.online }

func TestProjectScanHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	op, ts, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { op.Close(); ts.Close() })
	if _, err := op.Exec(`INSERT INTO nodes (id, name) VALUES (1, 'web')`); err != nil {
		t.Fatal(err)
	}

	agents := &fakeAgents{}
	h := NewProjectScanHandler(services.NewProjectScanService(context.Background(), db.NewRepo(op, ts), agents))
	r := gin.New()
	r.GET("/nodes/:id/project-settings", h.Get)
	r.PUT("/nodes/:id/project-settings", h.UpdateSettings)
	r.POST("/nodes/:id/rescan-projects", h.Rescan)
	r.POST("/nodes/:id/project-paths", h.AddPath)
	r.DELETE("/nodes/:id/project-paths/:pathId", h.DeletePath)

	call := func(method, path, body string) (int, string) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w.Code, w.Body.String()
	}
	expect := func(method, path, body string, wantCode int, wantBody string) {
		t.Helper()
		code, got := call(method, path, body)
		if code != wantCode || !strings.Contains(got, wantBody) {
			t.Fatalf("%s %s: got %d %s, want %d containing %s", method, path, code, got, wantCode, wantBody)
		}
	}

	expect("GET", "/nodes/9/project-settings", "", 404, "node not found")
	expect("GET", "/nodes/1/project-settings", "", 200,
		`{"node_id":1,"auto_scan":true,"scan_interval":900,"last_scanned_at":null,"errors":[],"paths":[],"online":false}`)

	for _, bad := range []string{`{"auto_scan":true,"scan_interval":30}`, `{"scan_interval":600}`, `not json`} {
		expect("PUT", "/nodes/1/project-settings", bad, 400, "scan_interval")
	}
	expect("PUT", "/nodes/1/project-settings", `{"auto_scan":false,"scan_interval":3600}`, 200, `"auto_scan":false,"scan_interval":3600`)
	if len(agents.pushed) != 1 || agents.pushed[0] != 1 {
		t.Fatalf("settings pushed to %v, want [1]", agents.pushed)
	}
	expect("PUT", "/nodes/9/project-settings", `{"auto_scan":true,"scan_interval":600}`, 404, "node not found")

	expect("POST", "/nodes/1/rescan-projects", "", 409, "offline")
	agents.online = true
	expect("POST", "/nodes/1/rescan-projects", "", 202, "rescan requested")

	// Manual paths: absolute only, cleaned, unique per node; each change is pushed.
	for _, bad := range []string{`{"path":"relative/app"}`, `{"path":""}`, `{}`} {
		expect("POST", "/nodes/1/project-paths", bad, 400, "absolute")
	}
	expect("POST", "/nodes/1/project-paths", `{"path":" /srv/app/../legacy/ "}`, 201, `{"id":1,"path":"/srv/legacy"`)
	expect("POST", "/nodes/1/project-paths", `{"path":"/srv/legacy"}`, 409, "already added")
	expect("POST", "/nodes/9/project-paths", `{"path":"/srv/x"}`, 404, "node not found")
	expect("GET", "/nodes/1/project-settings", "", 200, `"paths":[{"id":1,"path":"/srv/legacy"`)
	expect("DELETE", "/nodes/2/project-paths/1", "", 404, "project path not found")
	expect("DELETE", "/nodes/1/project-paths/1", "", 200, "removed")
	if len(agents.pushed) != 3 {
		t.Fatalf("%d pushes, want 3 (settings, add, delete)", len(agents.pushed))
	}
}
