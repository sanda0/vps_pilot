package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/internal/utils"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := utils.TokenValid(c)
		if err != nil {
			c.String(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}
		id, err := utils.ExtractTokenID(c)
		_ = id
		if err != nil {
			c.String(http.StatusUnauthorized, err.Error())
			c.Abort()
			return
		}
		c.Set("user_id", id)

		c.Next()
	}
}
