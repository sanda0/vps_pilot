package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
)

// newCronTestRepo has two projects: shop declares two cron jobs, blog none.
func newCronTestRepo(t *testing.T) *db.Repo {
	t.Helper()
	op, ts, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { op.Close(); ts.Close() })
	for _, q := range []string{
		`INSERT INTO nodes (id, name) VALUES (1, 'web')`,
		`INSERT INTO projects (id, node_id, name, path, cron) VALUES ('p1', 1, 'shop', '/var/www/shop',
		  '[{"command":"prune","schedule":"0 3 * * *"},{"command":"sitemap","schedule":"@hourly"}]')`,
		`INSERT INTO projects (id, node_id, name, path) VALUES ('p2', 1, 'blog', '/var/www/blog')`,
	} {
		if _, err := op.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return db.NewRepo(op, ts)
}

func TestProjectsShowTheirCronJobs(t *testing.T) {
	repo := newCronTestRepo(t)
	// A project without any has [].
	for id, want := range map[string]int{"p1": 2, "p2": 0} {
		p, _ := repo.Queries.GetProject(context.Background(), id)
		if got := dto.ConvertToProjectResponse(&p).Cron; got == nil || len(got) != want {
			t.Fatalf("project %s cron %+v", id, got)
		}
	}
}

// newCronTestServer adds three runs of shop's prune job, a day apart: exited,
// failed and skipped.
func newCronTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repo := newCronTestRepo(t)
	for i, status := range []string{"exited", "failed", "skipped"} {
		repo.Queries.InsertCronRun(context.Background(), db.InsertCronRunParams{
			ID: fmt.Sprintf("r%d", i), NodeID: 1, ProjectPath: "/var/www/shop", Command: "prune",
			Schedule: "0 3 * * *", ScheduledAt: 1793156400 + int64(i)*86400, Status: status,
			ExitCode: sql.NullInt64{Int64: int64(i), Valid: status != "skipped"}, Output: "out\n",
		})
	}
	h := NewCronHandler(services.NewCronService(context.Background(), repo))
	r := gin.New()
	r.GET("/projects/:id/cron", h.Jobs)
	return r
}

func TestCronJobsAPI(t *testing.T) {
	r := newCronTestServer(t)
	w := do(r, "GET", "/projects/p1/cron", "")
	body := w.Body.String()
	if w.Code != http.StatusOK || !strings.Contains(body, `"command":"prune","schedule":"0 3 * * *","last_run":{"id":"r2"`) ||
		!strings.Contains(body, `"status":"skipped","exit_code":null`) ||
		!strings.Contains(body, `"command":"sitemap","schedule":"@hourly","last_run":null`) {
		t.Fatalf("jobs: %d %s", w.Code, body)
	}
	if w := do(r, "GET", "/projects/p2/cron", ""); w.Code != http.StatusOK || w.Body.String() != `{"data":[]}` {
		t.Fatalf("no jobs: %d %s", w.Code, w.Body)
	}
	if w := do(r, "GET", "/projects/nope/cron", ""); w.Code != http.StatusNotFound {
		t.Fatalf("unknown project: %d %s", w.Code, w.Body)
	}
}
