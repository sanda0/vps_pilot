package app

import (
	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/common"
	"github.com/sanda0/vps_pilot/handlers"
	"github.com/sanda0/vps_pilot/middleware"
	"github.com/sanda0/vps_pilot/repositories"
	"github.com/sanda0/vps_pilot/services"
)

func Run(port string) {

	//init db
	conn := common.Conn{}
	db := conn.Connect()
	defer conn.Close()
	conn.Migrate()

	//init repositories
	userRepo := repositories.NewUserRepo(db)

	//init services
	userService := services.NewUserService(userRepo)

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
