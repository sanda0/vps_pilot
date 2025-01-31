package main

import (
	"flag"
	"log"

	"github.com/joho/godotenv"
	"github.com/sanda0/vps_pilot/cmd/app"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	port := flag.String("port", "8080", "port to listen on")
	createSuperuser := flag.Bool("create-superuser", false, "create superuser")
	flag.Parse()

	if *createSuperuser {
		// createSuperuser()
		return
	}

	app.Run(*port)

}
