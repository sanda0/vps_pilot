package db

import (
	"context"
	"database/sql"
	"log"
	"time"
)

const (
	retentionDays    = 7
	cleanupInterval  = 1 * time.Hour
	retentionSeconds = retentionDays * 24 * 60 * 60
)

// StartRetentionPolicyService starts a background service that deletes old time-series data
func StartRetentionPolicyService(ctx context.Context, timeseriesDB *sql.DB) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	// Run immediately on startup
	runRetentionCleanup(timeseriesDB)

	for {
		select {
		case <-ctx.Done():
			log.Println("Retention policy service stopped")
			return
		case <-ticker.C:
			runRetentionCleanup(timeseriesDB)
		}
	}
}

// runRetentionCleanup deletes data older than the retention period
func runRetentionCleanup(db *sql.DB) {
	cutoffTime := time.Now().Unix() - retentionSeconds

	// Clean system_stats table
	result, err := db.Exec("DELETE FROM system_stats WHERE timestamp < ?", cutoffTime)
	if err != nil {
		log.Printf("Error cleaning system_stats: %v\n", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("Retention cleanup: Deleted %d rows from system_stats\n", rowsAffected)
		}
	}

	// Clean net_stat table
	result, err = db.Exec("DELETE FROM net_stat WHERE timestamp < ?", cutoffTime)
	if err != nil {
		log.Printf("Error cleaning net_stat: %v\n", err)
	} else {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("Retention cleanup: Deleted %d rows from net_stat\n", rowsAffected)
		}
	}

	// Optional: Run VACUUM to reclaim space (can be expensive, consider running less frequently)
	// This is commented out by default - uncomment if you want automatic space reclamation
	// _, err = db.Exec("VACUUM")
	// if err != nil {
	// 	log.Printf("Error running VACUUM: %v\n", err)
	// }
}
