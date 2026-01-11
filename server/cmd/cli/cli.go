package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/utils"
)

func RunMigrations(dbPath string) error {
	if dbPath == "" {
		dbPath = "./data"
	}

	fmt.Println("Running migrations...")
	fmt.Printf("Database path: %s\n", dbPath)

	// Initialize databases (this runs migrations automatically)
	operationalDB, timeseriesDB, err := db.InitializeDatabases(dbPath)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	defer operationalDB.Close()
	defer timeseriesDB.Close()

	fmt.Println("✓ All migrations completed successfully")
	return nil
}

func CreateSuperuser(ctx context.Context, repo *db.Repo) {

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter email: ")
	email, _ := reader.ReadString('\n')

	fmt.Print("Enter password: ")
	password, _ := reader.ReadString('\n')

	fmt.Printf("Superuser created with email: %s and password: %s\n", email, password)

	hashedPassword, err := utils.HashString(strings.Trim(password, "\n"))
	if err != nil {
		fmt.Println("Error hashing password")
		return
	}

	user, err := repo.Queries.CreateUser(ctx, db.CreateUserParams{
		Email:        strings.Trim(email, "\n"),
		PasswordHash: string(hashedPassword),
		Username:     "admin",
	})
	if err != nil {
		fmt.Println("Error creating user")
		return
	}

	fmt.Println("User created with id: ", user.ID)

}

func CreateMakeFile() error {

	// Get the database path from the environment variable or use default
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data"
	}

	operationalDB := dbPath + "/operational.db"
	timeseriesDB := dbPath + "/timeseries.db"

	fileContent := `
# VPS Pilot Makefile
# Database path: ` + dbPath + `

# Run all migrations (uses built-in migration system)
migrate:
	go run main.go -migrate

# Generate SQLC code
sqlc:
	sqlc generate

# Build the server only (no UI)
build:
	go build -o vps_pilot main.go

# Build with embedded UI (requires frontend build first)
build-full:
	cd ../.. && ./build.sh

# Run the server
run:
	go run main.go

# Run the server with hot reload (requires air)
dev:
	air

# Create superuser
create-superuser:
	go run main.go -create-superuser

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f vps_pilot
	rm -f vps_pilot_server
	rm -f coverage.out
	rm -rf cmd/app/dist

# Show database info
db-info:
	@echo "Operational DB: ` + operationalDB + `"
	@echo "Timeseries DB: ` + timeseriesDB + `"
	@ls -lh ` + dbPath + ` 2>/dev/null || echo "Database directory not found"

# Backup databases
backup:
	@mkdir -p backups
	@timestamp=$$(date +%Y%m%d_%H%M%S); \
	cp ` + operationalDB + ` backups/operational_$$timestamp.db 2>/dev/null || true; \
	cp ` + timeseriesDB + ` backups/timeseries_$$timestamp.db 2>/dev/null || true; \
	echo "Backup created: backups/*_$$timestamp.db"

# Remove databases (use with caution!)
db-reset:
	@read -p "Are you sure you want to delete all databases? [y/N] " confirm; \
	if [ "$$confirm" = "y" ]; then \
		rm -f ` + operationalDB + ` ` + timeseriesDB + `; \
		echo "Databases deleted. Run 'make migrate' to recreate."; \
	else \
		echo "Cancelled."; \
	fi

.PHONY: migrate sqlc build build-full run dev create-superuser test test-coverage clean db-info backup db-reset

		`

	fmt.Println(fileContent)
	file, err := os.Create("makefile")
	if err != nil {
		return err
	}
	defer file.Close()
	bytes := []byte(fileContent)
	file.Write(bytes)

	return nil

}
