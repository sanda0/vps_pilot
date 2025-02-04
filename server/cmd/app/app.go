package app

import (
	"context"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/db"
	"github.com/sanda0/vps_pilot/handlers"
	"github.com/sanda0/vps_pilot/middleware"
	"github.com/sanda0/vps_pilot/services"
)

func Run(ctx context.Context, repo *db.Repo, port string) {

	//init services
	userService := services.NewUserService(ctx, repo)
	nodeService := services.NewNodeService(ctx, repo)

	//init handlers
	userHandler := handlers.NewAuthHandler(userService)
	nodeHander := handlers.NewNodeHandler(nodeService)

	server := gin.Default()
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Change to specific domains in production
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	//routes
	api := server.Group("/api/v1")

	//auth routes
	auth := api.Group("/auth")
	{
		auth.POST("/login", userHandler.Login)
	}

	//dashboard routes
	dashbaord := api.Group("/")
	dashbaord.Use(middleware.JwtAuthMiddleware())
	{
		dashbaord.GET("/profile", userHandler.Profile)
		nodes := dashbaord.Group("/nodes")
		{
			nodes.GET("", nodeHander.GetNodes)
		}
	}

	server.Run(":8000")
}
