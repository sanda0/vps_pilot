package db

import (
	"database/sql"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed sql/migrations/*.sql
var migrationsFS embed.FS

// Migration represents a database migration
type Migration struct {
	Version   string
	Name      string
	UpSQL     string
	DownSQL   string
	IsApplied bool
}

// DatabaseType represents the type of database
type DatabaseType string

const (
	OperationalDB DatabaseType = "operational"
	TimeseriesDB  DatabaseType = "timeseries"
)

// MigrationConfig defines which migrations belong to which database
type MigrationConfig struct {
	// Patterns that indicate a migration belongs to timeseries DB
	TimeseriesIncludePatterns []string
	// Patterns to exclude from operational DB
	OperationalExcludePatterns []string
}

// DefaultMigrationConfig returns the default migration separation configuration
var DefaultMigrationConfig = MigrationConfig{
	TimeseriesIncludePatterns: []string{
		"system_stat",
		"net_stat",
	},
	OperationalExcludePatterns: []string{
		"retention_policy",
		"enable_tablefunc",
		"system_stat",
		"net_stat",
	},
}

// belongsToTimeseries checks if a migration belongs to timeseries DB
func (c *MigrationConfig) belongsToTimeseries(migrationName string) bool {
	for _, pattern := range c.TimeseriesIncludePatterns {
		if strings.Contains(migrationName, pattern) {
			return true
		}
	}
	return false
}

// belongsToOperational checks if a migration belongs to operational DB
func (c *MigrationConfig) belongsToOperational(migrationName string) bool {
	for _, pattern := range c.OperationalExcludePatterns {
		if strings.Contains(migrationName, pattern) {
			return false
		}
	}
	return true
}

// RunMigrations executes all pending migrations on the given database
func RunMigrations(db *sql.DB, dbType string) error {
	return RunMigrationsWithConfig(db, dbType, &DefaultMigrationConfig)
}

// RunMigrationsWithConfig executes all pending migrations with a custom config
func RunMigrationsWithConfig(db *sql.DB, dbType string, config *MigrationConfig) error {
	// Create migrations table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migrations from embedded FS
	migrations, err := loadMigrations(dbType, config)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := getAppliedMigrations(db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Mark already applied migrations
	for i := range migrations {
		if _, exists := appliedMigrations[migrations[i].Version]; exists {
			migrations[i].IsApplied = true
		}
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.IsApplied {
			continue
		}

		fmt.Printf("Applying migration %s: %s\n", migration.Version, migration.Name)

		// Start transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Execute migration
		if _, err := tx.Exec(migration.UpSQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		// Record migration
		if _, err := tx.Exec("INSERT INTO migrations (version, name) VALUES (?, ?)", migration.Version, migration.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Version, err)
		}

		fmt.Printf("Migration %s applied successfully\n", migration.Version)
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			applied_at INTEGER DEFAULT (strftime('%s', 'now'))
		)
	`
	_, err := db.Exec(query)
	return err
}

// loadMigrations loads all migrations from embedded filesystem
func loadMigrations(dbType string, config *MigrationConfig) ([]Migration, error) {
	entries, err := migrationsFS.ReadDir("sql/migrations")
	if err != nil {
		return nil, err
	}

	// Group migrations by version
	migrationMap := make(map[string]*Migration)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		// Parse filename: 20250202111257_user_table.up.sql
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		isUp := strings.HasSuffix(filename, ".up.sql")
		isDown := strings.HasSuffix(filename, ".down.sql")

		if !isUp && !isDown {
			continue
		}

		// Extract name (everything between version and .up/.down.sql)
		nameEnd := strings.LastIndex(filename, ".up.sql")
		if nameEnd == -1 {
			nameEnd = strings.LastIndex(filename, ".down.sql")
		}
		name := filename[len(version)+1 : nameEnd]

		// Read file content
		content, err := migrationsFS.ReadFile(filepath.Join("sql/migrations", filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// Get or create migration
		migration, exists := migrationMap[version]
		if !exists {
			migration = &Migration{
				Version: version,
				Name:    name,
			}
			migrationMap[version] = migration
		}

		// Set SQL content
		if isUp {
			migration.UpSQL = string(content)
		} else {
			migration.DownSQL = string(content)
		}
	}

	// Convert map to sorted slice
	var migrations []Migration
	for _, migration := range migrationMap {
		// Only include migrations with up SQL
		if migration.UpSQL != "" {
			migrations = append(migrations, *migration)
		}
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Filter migrations based on dbType using config
	var filteredMigrations []Migration
	for _, migration := range migrations {
		if dbType == string(TimeseriesDB) {
			// Only include migrations that belong to timeseries
			if config.belongsToTimeseries(migration.Name) {
				filteredMigrations = append(filteredMigrations, migration)
			}
		} else if dbType == string(OperationalDB) {
			// Include migrations that belong to operational
			if config.belongsToOperational(migration.Name) {
				filteredMigrations = append(filteredMigrations, migration)
			}
		}
	}

	return filteredMigrations, nil
}

// getAppliedMigrations returns a set of already applied migration versions
func getAppliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}
