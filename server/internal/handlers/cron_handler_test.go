package handlers

import (
	"context"
	"testing"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
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
