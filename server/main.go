package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/sanda0/vps_pilot/cmd/app"
	"github.com/sanda0/vps_pilot/cmd/cli"
	"github.com/sanda0/vps_pilot/db"
	tcpserver "github.com/sanda0/vps_pilot/tcp_server"

	_ "github.com/lib/pq"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	//init db
	con, err := sql.Open("postgres", fmt.Sprintf("dbname=%s password=%s user=%s host=%s sslmode=require",
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_HOST"),
	))

	if err != nil {
		panic(err)
	}

	//init ctx
	ctx := context.Background()
	repo := db.NewRepo(con)

	//init tcp server
	go tcpserver.StartTcpServer(ctx, repo, "55001")

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

	app.Run(ctx, repo, *port)

}
