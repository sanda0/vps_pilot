package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// InitializeDatabases creates and initializes both operational and timeseries SQLite databases
func InitializeDatabases(dbDir string) (*sql.DB, *sql.DB, error) {
	// Create database directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Initialize operational database
	operationalPath := filepath.Join(dbDir, "operational.db")
	operationalDB, err := initSQLiteDB(operationalPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize operational database: %w", err)
	}

	// Initialize timeseries database
	timeseriesPath := filepath.Join(dbDir, "timeseries.db")
	timeseriesDB, err := initSQLiteDB(timeseriesPath)
	if err != nil {
		operationalDB.Close()
		return nil, nil, fmt.Errorf("failed to initialize timeseries database: %w", err)
	}

	// Run migrations
	fmt.Println("Running migrations for operational database...")
	if err := RunMigrations(operationalDB, "operational"); err != nil {
		operationalDB.Close()
		timeseriesDB.Close()
		return nil, nil, fmt.Errorf("failed to run operational migrations: %w", err)
	}

	fmt.Println("Running migrations for timeseries database...")
	if err := RunMigrations(timeseriesDB, "timeseries"); err != nil {
		operationalDB.Close()
		timeseriesDB.Close()
		return nil, nil, fmt.Errorf("failed to run timeseries migrations: %w", err)
	}

	fmt.Println("Databases initialized successfully")
	return operationalDB, timeseriesDB, nil
}

// initSQLiteDB initializes a single SQLite database with optimized settings
func initSQLiteDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", path, err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database %s: %w", path, err)
	}

	// Configure SQLite pragmas for optimal performance
	pragmas := []string{
		"PRAGMA foreign_keys = ON",         // Enable foreign key constraints
		"PRAGMA journal_mode = WAL",        // Write-Ahead Logging for better concurrency
		"PRAGMA synchronous = NORMAL",      // Balance between safety and speed
		"PRAGMA cache_size = -64000",       // 64MB cache
		"PRAGMA busy_timeout = 5000",       // 5 second timeout for locked database
		"PRAGMA temp_store = MEMORY",       // Store temp tables in memory
		"PRAGMA mmap_size = 268435456",     // 256MB memory-mapped I/O
		"PRAGMA page_size = 4096",          // Optimal page size
		"PRAGMA auto_vacuum = INCREMENTAL", // Incremental auto-vacuum
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute pragma %s: %w", pragma, err)
		}
	}

	// Set connection pool limits
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return db, nil
}
