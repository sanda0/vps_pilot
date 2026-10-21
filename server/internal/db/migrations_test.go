package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// The agent_enrollment migration rebuilds the nodes table. Child rows must
// survive it (DROP TABLE with foreign keys on would cascade-delete them).
func TestAgentEnrollmentMigrationKeepsChildRows(t *testing.T) {
	sqlDB, err := initSQLiteDB(filepath.Join(t.TempDir(), "operational.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	if err := createMigrationsTable(sqlDB); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations(string(OperationalDB), &DefaultMigrationConfig)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		if m.Version >= "20261020000002" {
			break
		}
		if err := applyMigration(sqlDB, m); err != nil {
			t.Fatal(err)
		}
	}

	mustExec(t, sqlDB, `INSERT INTO nodes (id, name, ip) VALUES (1, 'web', '10.0.0.1')`)
	mustExec(t, sqlDB, `INSERT INTO node_sys_info (node_id, os) VALUES (1, 'linux')`)
	mustExec(t, sqlDB, `INSERT INTO projects (node_id, name, path) VALUES (1, 'app', '/srv/app')`)

	if err := RunMigrations(sqlDB, string(OperationalDB)); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{"nodes", "node_sys_info", "projects"} {
		var n int
		if err := sqlDB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("%s: got %d rows after migration, want 1", table, n)
		}
	}

	// ip is no longer unique: two nodes behind the same NAT are fine.
	mustExec(t, sqlDB, `INSERT INTO nodes (name, ip, agent_id) VALUES ('web2', '10.0.0.1', 'a')`)

	// Foreign keys are still enforced on pooled connections afterwards.
	if _, err := sqlDB.Exec(`INSERT INTO node_sys_info (node_id, os) VALUES (999, 'linux')`); err == nil {
		t.Error("expected foreign key violation for unknown node")
	}
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()
	if _, err := db.Exec(query); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
}
