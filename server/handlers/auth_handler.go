package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/dto"
	"github.com/sanda0/vps_pilot/services"
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

	c.JSON(200, gin.H{"data": user})
}

// Login implements AuthHandler.
func (a *authHandler) Login(c *gin.Context) {
	form := dto.UserLoginDto{}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.SetCookie("__tkn__", "", -1, "/", "", true, true)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userResponse, err := a.userService.Login(form)
	if err != nil {
		c.SetCookie("__tkn__", "", -1, "/", "", true, true)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.SetCookie("__tkn__", userResponse.Token, 3600, "/", "", true, true)

	c.JSON(200, gin.H{"data": userResponse})

}

func NewAuthHandler(userService services.UserService) AuthHandler {
	return &authHandler{
		userService: userService,
	}
}
