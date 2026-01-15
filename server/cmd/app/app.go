package app

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/handlers"
	"github.com/sanda0/vps_pilot/internal/middleware"
	"github.com/sanda0/vps_pilot/internal/services"
)

//go:embed all:dist
var staticFiles embed.FS

// hasEmbeddedUI checks if the UI files are embedded
func hasEmbeddedUI() bool {
	_, err := staticFiles.ReadDir("dist")
	return err == nil
}

func Run(ctx context.Context, repo *db.Repo, port string) {

	//init services
	userService := services.NewUserService(ctx, repo)
	nodeService := services.NewNodeService(ctx, repo)
	alertService := services.NewAlertService(ctx, repo)
	projectService := services.NewProjectService(repo, ctx)

	//init handlers
	userHandler := handlers.NewAuthHandler(userService)
	nodeHander := handlers.NewNodeHandler(nodeService)
	alertHandler := handlers.NewAlertHandler(alertService)
	projectHandler := handlers.NewProjectHandler(projectService)
	githubHandler := handlers.NewGitHubHandler(userService)

	server := gin.Default()

	// CORS configuration - allow frontend origin in development
	server.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:8000"}, // Development + embedded
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	//routes
	api := server.Group("/api/v1")
	// api.

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

		// GitHub integration routes
		github := dashbaord.Group("/github")
		{
			github.POST("/token", githubHandler.SaveToken)
			github.GET("/repos", githubHandler.GetRepos)
			github.GET("/status", githubHandler.GetStatus)
			github.DELETE("/token", githubHandler.DeleteToken)
		}

		nodes := dashbaord.Group("/nodes")
		{
			nodes.GET("", nodeHander.GetNodes)
			nodes.PUT("/change-name", nodeHander.UpdateName)
			nodes.GET("/:id", nodeHander.GetNode)
			nodes.GET("/ws/system-stat", nodeHander.SystemStatWSHandler)
			nodes.GET("/:id/projects", projectHandler.ListProjectsByNode)
		}
		alerts := dashbaord.Group("/alerts")
		{
			alerts.GET("/:id", alertHandler.GetAlert)
			alerts.POST("", alertHandler.CreateAlert)
			alerts.GET("", alertHandler.GetAlerts)
			alerts.PUT("/activate", alertHandler.ActivateAlert)
			alerts.PUT("/deactivate", alertHandler.DeactivateAlert)
			alerts.DELETE("/:id", alertHandler.DeleteAlert)
			alerts.PUT("", alertHandler.UpdateAlert)
		}
		projects := dashbaord.Group("/projects")
		{
			projects.POST("", projectHandler.CreateProject)
			projects.GET("", projectHandler.ListProjects)
			projects.GET("/:id", projectHandler.GetProject)
			projects.PUT("/:id", projectHandler.UpdateProject)
			projects.DELETE("/:id", projectHandler.DeleteProject)
		}
	}

	// Serve embedded static files
	serveEmbeddedFiles(server)

	server.Run(":8000")
}

// serveEmbeddedFiles serves the embedded frontend files
func serveEmbeddedFiles(router *gin.Engine) {
	// Try to get the dist subdirectory from embedded FS
	distFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		log.Println("Warning: Failed to load embedded UI files. Frontend will not be available.")
		log.Println("Run 'make build-ui' to build and embed the frontend.")
		return
	}

	// Serve static files (JS, CSS, images, etc.)
	router.GET("/assets/*filepath", func(c *gin.Context) {
		filePath := strings.TrimPrefix(c.Param("filepath"), "/")
		filePath = "assets/" + filePath
		data, err := fs.ReadFile(distFS, filePath)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		// Set appropriate content type
		contentType := getContentType(filePath)
		c.Data(http.StatusOK, contentType, data)
	})

	// Serve favicon and other root files
	router.GET("/favicon.ico", func(c *gin.Context) {
		data, err := fs.ReadFile(distFS, "favicon.ico")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "image/x-icon", data)
	})

	// SPA fallback - serve index.html for all other routes (client-side routing)
	router.NoRoute(func(c *gin.Context) {
		// Don't serve index.html for API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// Serve index.html for all other routes (SPA)
		data, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to load frontend")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	log.Println("✓ Embedded UI loaded successfully")
}

// getContentType returns the appropriate content type based on file extension
func getContentType(filename string) string {
	if strings.HasSuffix(filename, ".js") {
		return "application/javascript"
	} else if strings.HasSuffix(filename, ".css") {
		return "text/css"
	} else if strings.HasSuffix(filename, ".json") {
		return "application/json"
	} else if strings.HasSuffix(filename, ".png") {
		return "image/png"
	} else if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		return "image/jpeg"
	} else if strings.HasSuffix(filename, ".svg") {
		return "image/svg+xml"
	} else if strings.HasSuffix(filename, ".woff") {
		return "font/woff"
	} else if strings.HasSuffix(filename, ".woff2") {
		return "font/woff2"
	}
	return "application/octet-stream"
}
