package main

import (
	"flag"
	"log"

	"github.com/joho/godotenv"
	"github.com/sanda0/vps_pilot/cmd/app"
	"github.com/sanda0/vps_pilot/cmd/cli"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// conn := common.Conn{}
	// conn.Migrate()

	port := flag.String("port", "8080", "port to listen on")
	createSuperuser := flag.Bool("create-superuser", false, "create superuser")
	createMakefile := flag.Bool("create-makefile", false, "create makefile")
	flag.Parse()

	if *createSuperuser {
		cli.CreateSuperuser()
		return
	}

	if *createMakefile {
		err := cli.CreateMakeFile()
		if err != nil {
			log.Fatal("Error creating Makefile")
		}
		return
	}

	app.Run(*port)

}
