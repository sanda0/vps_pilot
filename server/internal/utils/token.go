package utils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

func ExtractTokenFromHeader(c *gin.Context) string {
	// get token from cookie
	cookie, err := c.Request.Cookie("__tkn__")

	if err == nil {
		return cookie.Value
	}

	return ""

}

// func ExtractTokenFromHeader(c *gin.Context) string {
// 	bearerToken := c.Request.Header.Get("Authorization")
// 	if len(bearerToken) > 7 && bearerToken[:7] == "Bearer " {
// 		return bearerToken[7:]
// 	}
// 	return ""
// }

func GenerateToken(user_id int32) (string, error) {
	token_lifespan, err := strconv.Atoi(os.Getenv("TOKEN_LIFESPAN")) // in minutes
	if err != nil {
		log.Println("token life span error: ", err)
		return "", err
	}

	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = user_id
	claims["exp"] = time.Now().Add(time.Minute * time.Duration(token_lifespan)).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(os.Getenv("TOKEN_SECRET")))

}

func TokenValid(c *gin.Context) error {
	tokenString := ExtractTokenFromHeader(c)
	_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("TOKEN_SECRET")), nil
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
		return []byte(os.Getenv("TOKEN_SECRET")), nil
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

func WriteTokenToCookie(c *gin.Context, token string) {
	//set http only cookie
	cookie := &http.Cookie{
		Name:     "__tkn__",
		Value:    token,
		HttpOnly: true,
		Secure:   false, //TODO: change to true in production
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(c.Writer, cookie)

}
