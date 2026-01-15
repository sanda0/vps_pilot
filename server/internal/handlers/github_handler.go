package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/services"
)

type GitHubHandler interface {
	SaveToken(c *gin.Context)
	GetRepos(c *gin.Context)
	GetStatus(c *gin.Context)
	DeleteToken(c *gin.Context)
}

type githubHandler struct {
	userService services.UserService
}

type GitHubRepo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Private       bool   `json:"private"`
	HTMLURL       string `json:"html_url"`
	CloneURL      string `json:"clone_url"`
	SSHURL        string `json:"ssh_url"`
	Description   string `json:"description"`
	DefaultBranch string `json:"default_branch"`
	UpdatedAt     string `json:"updated_at"`
}

func NewGitHubHandler(userService services.UserService) GitHubHandler {
	return &githubHandler{
		userService: userService,
	}
}

// SaveToken saves the GitHub personal access token
func (h *githubHandler) SaveToken(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Token is required",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// Test the token first
	repos, err := h.fetchRepos(req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid GitHub token. Please check your token and try again.",
		})
		return
	}

	// Save token
	err = h.userService.SaveGitHubToken(userID.(int32), req.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save GitHub token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "GitHub token saved successfully",
		"repos_count": len(repos),
	})
}

// GetRepos fetches repositories from GitHub
func (h *githubHandler) GetRepos(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// Get token
	token, err := h.userService.GetGitHubToken(userID.(int32))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "GitHub token not found. Please connect your GitHub account first.",
			"connected": false,
		})
		return
	}

	// Fetch repos
	repos, err := h.fetchRepos(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch repositories from GitHub",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": repos,
	})
}

// GetStatus checks if GitHub is connected
func (h *githubHandler) GetStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	_, err := h.userService.GetGitHubToken(userID.(int32))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"connected": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
	})
}

// DeleteToken removes the GitHub token
func (h *githubHandler) DeleteToken(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	err := h.userService.RemoveGitHubToken(userID.(int32))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to disconnect GitHub",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "GitHub disconnected successfully",
	})
}

// fetchRepos fetches repositories from GitHub API
func (h *githubHandler) fetchRepos(token string) ([]GitHubRepo, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.github.com/user/repos?per_page=100&sort=updated", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}
