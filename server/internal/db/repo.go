package db

import "database/sql"

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
