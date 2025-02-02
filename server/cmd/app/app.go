package app

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/db"
	"github.com/sanda0/vps_pilot/handlers"
	"github.com/sanda0/vps_pilot/middleware"
	"github.com/sanda0/vps_pilot/services"
)

func Run(ctx context.Context, repo *db.Repo, port string) {

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
