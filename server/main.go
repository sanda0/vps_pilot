package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/sanda0/vps_pilot/cmd/app"
	"github.com/sanda0/vps_pilot/cmd/cli"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: Error loading .env file, using environment variables")
	}

	// Get database directory from environment or use default
	dbDir := os.Getenv("DB_PATH")
	if dbDir == "" {
		// Use current directory by default
		dbDir = "./data"
	}

	// Expand home directory if present
	if dbDir[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to get home directory:", err)
		}
		dbDir = filepath.Join(homeDir, dbDir[1:])
	}

	// Initialize databases
	log.Printf("Initializing databases in: %s\n", dbDir)
	operationalDB, timeseriesDB, err := db.InitializeDatabases(dbDir)
	if err != nil {
		log.Fatal("Failed to initialize databases:", err)
	}
	defer operationalDB.Close()
	defer timeseriesDB.Close()

	//init ctx
	ctx := context.Background()
	repo := db.NewRepo(operationalDB, timeseriesDB)

	//start retention policy service
	go db.StartRetentionPolicyService(ctx, timeseriesDB)

	port := flag.String("port", "8080", "port to listen on")
	createSuperuser := flag.Bool("create-superuser", false, "create superuser")
	createMakefile := flag.Bool("create-makefile", false, "create makefile")
	flag.Parse()

	if *createSuperuser {
		cli.CreateSuperuser(ctx, repo)
		return
	}

	if *createMakefile {
		err := cli.CreateMakeFile()
		if err != nil {
			log.Fatal("Error creating Makefile")
		}
		return
	}

	//init tcp server
	go tcpserver.StartTcpServer(ctx, repo, "55001")

	app.Run(ctx, repo, *port)

}
