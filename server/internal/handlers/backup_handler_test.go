package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

type fakeBackupper struct {
	repo    *db.Repo
	err     error
	started []tcpserver.StartBackup
}

func (f *fakeBackupper) Start(ctx context.Context, req tcpserver.StartBackup) (db.BackupRun, error) {
	if f.err != nil {
		return db.BackupRun{}, f.err
	}
	f.started = append(f.started, req)
	return f.repo.Queries.CreateBackupRun(ctx, db.CreateBackupRunParams{
		ID: "b1", NodeID: req.NodeID, ProjectPath: req.ProjectPath, Username: req.Username,
		UserID: sql.NullInt64{Int64: req.UserID, Valid: true},
	})
}

func newBackupTestServer(t *testing.T) (*gin.Engine, *fakeBackupper, *db.Repo) {
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
		`INSERT INTO projects (id, node_id, name, path, backups) VALUES ('p1', 1, 'shop', '/var/www/shop', '{"local_dir":"../backups"}')`,
		`INSERT INTO projects (id, node_id, name, path) VALUES ('p2', 1, 'blog', '/var/www/blog')`,
	} {
		if _, err := op.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	repo := db.NewRepo(op, ts)
	backups := &fakeBackupper{repo: repo}
	h := NewBackupHandler(services.NewBackupService(context.Background(), repo, backups))
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", int32(1)) })
	r.POST("/projects/:id/backups", h.Start)
	r.GET("/projects/:id/backups", h.List)
	return r, backups, repo
}

func TestBackupAPI(t *testing.T) {
	r, backups, repo := newBackupTestServer(t)
	for path, want := range map[string]int{"/projects/nope/backups": http.StatusNotFound, "/projects/p2/backups": http.StatusConflict} {
		if w := do(r, "POST", path, ""); w.Code != want {
			t.Errorf("POST %s: %d %s, want %d", path, w.Code, w.Body, want)
		}
	}
	w := do(r, "POST", "/projects/p1/backups", "")
	if w.Code != http.StatusAccepted || !strings.Contains(w.Body.String(), `"status":"running"`) {
		t.Fatalf("start: %d %s", w.Code, w.Body)
	}
	if got := backups.started; len(got) != 1 || got[0] != (tcpserver.StartBackup{NodeID: 1, ProjectPath: "/var/www/shop", UserID: 1, Username: "admin"}) {
		t.Fatalf("started %+v", got)
	}

	// A scheduled one the agent reported, newer than the manual one.
	repo.Queries.FinishBackupRun(context.Background(), db.FinishBackupRunParams{
		ID: "s1", NodeID: 1, ProjectPath: "/var/www/shop", TriggerKind: "schedule", Status: "ok",
		File: "shop-20261031T030000Z.zip", Size: 42, UploadStatus: "skipped", StartedAt: 1 << 40,
	})
	w = do(r, "GET", "/projects/p1/backups", "")
	body := w.Body.String()
	if w.Code != http.StatusOK || strings.Index(body, `"id":"s1"`) > strings.Index(body, `"id":"b1"`) ||
		!strings.Contains(body, `"trigger":"schedule"`) || !strings.Contains(body, `"username":"admin"`) {
		t.Fatalf("list: %d %s", w.Code, body)
	}
	if w := do(r, "GET", "/projects/p2/backups", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"data":[]`) {
		t.Fatalf("empty list: %d %s", w.Code, w.Body)
	}

	for err, want := range map[error]int{tcpserver.ErrNodeOffline: http.StatusConflict, tcpserver.ErrNoBackups: http.StatusConflict} {
		backups.err = err
		if w := do(r, "POST", "/projects/p1/backups", ""); w.Code != want || !strings.Contains(w.Body.String(), err.Error()) {
			t.Errorf("%v: %d %s", err, w.Code, w.Body)
		}
	}
}
