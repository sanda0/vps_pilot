package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repo struct {
	*Queries
	OperationalDB     *sql.DB
	TimeseriesDB      *sql.DB
	TimeseriesQueries *Queries // Queries for timeseries database
}

func NewRepo(operationalDB, timeseriesDB *sql.DB) *Repo {
	return &Repo{
		OperationalDB:     operationalDB,
		TimeseriesDB:      timeseriesDB,
		Queries:           New(operationalDB), // Default to operational DB for SQLC queries
		TimeseriesQueries: New(timeseriesDB),  // Queries for timeseries DB
	}
}

// SaveGitHubToken saves the GitHub token for a user
func (r *Repo) SaveGitHubToken(ctx context.Context, userID int32, token string) error {
	query := `UPDATE users SET github_token = ?, updated_at = ? WHERE id = ?`
	_, err := r.OperationalDB.ExecContext(ctx, query, token, time.Now().Unix(), userID)
	return err
}

// GetGitHubToken retrieves the GitHub token for a user
func (r *Repo) GetGitHubToken(ctx context.Context, userID int32) (string, error) {
	var token sql.NullString
	query := `SELECT github_token FROM users WHERE id = ?`
	err := r.OperationalDB.QueryRowContext(ctx, query, userID).Scan(&token)
	if err != nil {
		return "", err
	}
	if !token.Valid || token.String == "" {
		return "", fmt.Errorf("GitHub token not found")
	}
	return token.String, nil
}

// RemoveGitHubToken removes the GitHub token for a user
func (r *Repo) RemoveGitHubToken(ctx context.Context, userID int32) error {
	query := `UPDATE users SET github_token = NULL, updated_at = ? WHERE id = ?`
	_, err := r.OperationalDB.ExecContext(ctx, query, time.Now().Unix(), userID)
	return err
}
