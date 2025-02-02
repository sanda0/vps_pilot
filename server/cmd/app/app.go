package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/db"
	"github.com/sanda0/vps_pilot/handlers"
	"github.com/sanda0/vps_pilot/middleware"
	"github.com/sanda0/vps_pilot/services"
)

func Run(port string) {

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

	//init services
	userService := services.NewUserService(ctx, repo)

	//init handlers
	userHandler := handlers.NewAuthHandler(userService)

	server := gin.Default()
	server.Use(middleware.CORSMiddleware())

	//routes
	api := server.Group("/api/v1")

	auth := api.Group("/auth")
	{
		auth.POST("/login", userHandler.Login)
	}

	dashbaord := api.Group("/")
	dashbaord.Use(middleware.JwtAuthMiddleware())
	{
		dashbaord.GET("/profile", userHandler.Profile)
	}

	server.Run(":8000")
}
