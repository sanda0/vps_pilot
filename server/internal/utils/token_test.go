package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func contextWithToken(token string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.AddCookie(&http.Cookie{Name: "__tkn__", Value: token})
	return c
}

func sign(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, key any) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestTokenRoundTrip(t *testing.T) {
	t.Setenv("TOKEN_SECRET", "test-secret")
	token, err := GenerateToken(42)
	if err != nil {
		t.Fatal(err)
	}
	c := contextWithToken(token)
	if err := TokenValid(c); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	if id, err := ExtractTokenID(c); err != nil || id != 42 {
		t.Fatalf("got user %d, err %v; want 42", id, err)
	}
}

func TestTokenRejected(t *testing.T) {
	t.Setenv("TOKEN_SECRET", "test-secret")
	exp := time.Now().Add(time.Hour).Unix()
	for name, token := range map[string]string{
		"expired":      sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1, "exp": time.Now().Add(-time.Minute).Unix()}, []byte("test-secret")),
		"wrong secret": sign(t, jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 1, "exp": exp}, []byte("other-secret")),
		"alg none":     sign(t, jwt.SigningMethodNone, jwt.MapClaims{"user_id": 1, "exp": exp}, jwt.UnsafeAllowNoneSignatureType),
		"garbage":      "not-a-jwt",
	} {
		if err := TokenValid(contextWithToken(token)); err == nil {
			t.Errorf("%s: token accepted", name)
		}
	}
}
