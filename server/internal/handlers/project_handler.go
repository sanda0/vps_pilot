package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
)

type ProjectHandler interface {
	// Agent-facing
	AgentSyncProject(c *gin.Context)
	AgentBulkSyncProjects(c *gin.Context)

	// Dashboard-facing
	GetProject(c *gin.Context)
	ListProjects(c *gin.Context)
	ListProjectsByNode(c *gin.Context)
	DeleteProject(c *gin.Context)
}

type projectHandler struct {
	projectService services.ProjectService
}

func NewProjectHandler(projectService services.ProjectService) ProjectHandler {
	return &projectHandler{
		projectService: projectService,
	}
}

// AgentSyncProject handles POST /api/v1/agent/projects/sync
// Called by the agent when it discovers or updates a single project config.
func (h *projectHandler) AgentSyncProject(c *gin.Context) {
	var req dto.AgentProjectSyncRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	project, err := h.projectService.UpsertFromAgent(&req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "node not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"error":   "Failed to sync project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, project)
}

// AgentBulkSyncProjects handles POST /api/v1/agent/projects/bulk-sync
// Called by the agent to report all projects found on a node in one request.
func (h *projectHandler) AgentBulkSyncProjects(c *gin.Context) {
	var req dto.AgentProjectsBulkSyncRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	projects, err := h.projectService.BulkSyncFromAgent(&req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "node not found" {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{
			"error":   "Failed to bulk sync projects",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"synced": len(projects),
		"data":   projects,
	})
}

// GetProject handles GET /api/v1/projects/:id
func (h *projectHandler) GetProject(c *gin.Context) {
	id := c.Param("id")

	project, err := h.projectService.GetProject(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Project not found",
		})
		return
	}

	c.JSON(http.StatusOK, project)
}

// ListProjects handles GET /api/v1/projects
func (h *projectHandler) ListProjects(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	projects, err := h.projectService.ListProjects(int32(limit), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list projects",
			"details": err.Error(),
		})
		return
	}

	total, err := h.projectService.CountProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to count projects",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   projects,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ListProjectsByNode handles GET /api/v1/nodes/:id/projects
func (h *projectHandler) ListProjectsByNode(c *gin.Context) {
	nodeIDStr := c.Param("id")
	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid node ID",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	projects, err := h.projectService.ListProjectsByNode(int32(nodeID), int32(limit), int32(offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list projects",
			"details": err.Error(),
		})
		return
	}

	total, err := h.projectService.CountProjectsByNode(int32(nodeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to count projects",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   projects,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// DeleteProject handles DELETE /api/v1/projects/:id
func (h *projectHandler) DeleteProject(c *gin.Context) {
	id := c.Param("id")

	if err := h.projectService.DeleteProject(id); err != nil {
		if err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Project not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete project",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Project deleted successfully",
	})
}
