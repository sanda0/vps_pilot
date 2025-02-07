package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/utils"
)

type AuthHandler interface {
	Login(c *gin.Context)
	Profile(c *gin.Context)
}

type authHandler struct {
	userService services.UserService
}

// Profile implements AuthHandler.
func (a *authHandler) Profile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	user, err := a.userService.Profile(userID.(int32))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userResponse := dto.UserLoginResponseDto{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}

	c.JSON(200, gin.H{"data": userResponse})
}

// Login implements AuthHandler.
func (a *authHandler) Login(c *gin.Context) {
	form := dto.UserLoginDto{}
	if err := c.ShouldBindJSON(&form); err != nil {
		utils.WriteTokenToCookie(c, "")
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userResponse, err := a.userService.Login(form)
	if err != nil {
		utils.WriteTokenToCookie(c, "")
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	token, err := utils.GenerateToken(userResponse.ID)
	if err != nil {
		utils.WriteTokenToCookie(c, "")
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	utils.WriteTokenToCookie(c, token)

	c.JSON(200, gin.H{"data": userResponse})

}

func NewAuthHandler(userService services.UserService) AuthHandler {
	return &authHandler{
		userService: userService,
	}
}
