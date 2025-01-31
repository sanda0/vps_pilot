package main

import (
	"flag"
	"log"

	"github.com/joho/godotenv"
	"github.com/sanda0/vps_pilot/cmd/app"
	"github.com/sanda0/vps_pilot/cmd/cli"
	"github.com/sanda0/vps_pilot/common"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	conn := common.Conn{}
	conn.Migrate()

	port := flag.String("port", "8080", "port to listen on")
	createSuperuser := flag.Bool("create-superuser", false, "create superuser")
	flag.Parse()

	if *createSuperuser {
		cli.CreateSuperuser()
		return
	}

	app.Run(*port)

}
