package utils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func ExtractTokenFromHeader(c *gin.Context) string {
	// get token from cookie
	cookie, err := c.Request.Cookie("__tkn__")

	if err == nil {
		return cookie.Value
	}

	return ""

}

const (
	defaultTokenLifespan = 60 // minutes
	defaultTokenSecret   = "default-secret-change-in-production"
)

// tokenLifespan returns TOKEN_LIFESPAN in minutes, or the default when it is
// unset or not an integer. It does not log; see WarnTokenConfig.
func tokenLifespan() int {
	if lifespan, err := strconv.Atoi(os.Getenv("TOKEN_LIFESPAN")); err == nil {
		return lifespan
	}
	return defaultTokenLifespan
}

// WarnTokenConfig logs once, at hub startup, when TOKEN_LIFESPAN or TOKEN_SECRET
// fall back to their defaults.
func WarnTokenConfig() {
	if lifespanStr := os.Getenv("TOKEN_LIFESPAN"); lifespanStr == "" {
		log.Printf("Warning: TOKEN_LIFESPAN not set, using default %d minutes\n", defaultTokenLifespan)
	} else if _, err := strconv.Atoi(lifespanStr); err != nil {
		log.Printf("Warning: Invalid TOKEN_LIFESPAN value '%s', using default %d minutes\n", lifespanStr, defaultTokenLifespan)
	}
	if os.Getenv("TOKEN_SECRET") == "" {
		log.Println("Warning: TOKEN_SECRET not set, using default (INSECURE - set TOKEN_SECRET in production)")
	}
}

func GenerateToken(user_id int32) (string, error) {
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = user_id
	claims["exp"] = time.Now().Add(time.Minute * time.Duration(tokenLifespan())).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(getTokenSecret()))

}

func getTokenSecret() string {
	if tokenSecret := os.Getenv("TOKEN_SECRET"); tokenSecret != "" {
		return tokenSecret
	}
	return defaultTokenSecret
}

func TokenValid(c *gin.Context) error {
	tokenString := ExtractTokenFromHeader(c)
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(getTokenSecret()), nil
	})
	if err != nil {

		return err
	}
	return nil
}

func ExtractTokenID(c *gin.Context) (int32, error) {

	tokenString := ExtractTokenFromHeader(c)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(getTokenSecret()), nil
	})
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		user_id := fmt.Sprintf("%v", claims["user_id"])
		userID, err := strconv.ParseUint(user_id, 10, 64)
		if err != nil {
			return 0, err
		}
		return int32(userID), nil
	}
	return 0, nil
}

// cookieSecure decides the cookie Secure flag. COOKIE_SECURE=true/false forces
// it; when unset, it follows whether the request arrived over HTTPS (directly
// or through a reverse proxy setting X-Forwarded-Proto).
func cookieSecure(c *gin.Context) bool {
	if v, err := strconv.ParseBool(os.Getenv("COOKIE_SECURE")); err == nil {
		return v
	}
	return c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func WriteTokenToCookie(c *gin.Context, token string) {
	//set http only cookie
	cookie := &http.Cookie{
		Name:     "__tkn__",
		Value:    token,
		HttpOnly: true,
		Secure:   cookieSecure(c),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(c.Writer, cookie)

}
