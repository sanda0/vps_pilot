package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLogoutClearsSessionCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/auth/logout", NewAuthHandler(nil).Logout)

	for name, cookieHeader := range map[string]string{
		"without a session cookie": "",
		"with a session cookie":    "__tkn__=stale-token; theme=dark",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/auth/logout", nil)
			if cookieHeader != "" {
				req.Header.Set("Cookie", cookieHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != 200 {
				t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "logged out") {
				t.Fatalf("body = %s, want it to mention logged out", w.Body.String())
			}

			var cleared bool
			for _, c := range w.Result().Cookies() {
				if c.Name != "__tkn__" {
					continue
				}
				cleared = true
				if c.Value != "" {
					t.Fatalf("__tkn__ = %q, want it emptied", c.Value)
				}
				if !c.HttpOnly {
					t.Fatal("__tkn__ must stay HttpOnly")
				}
				if c.Path != "/" {
					t.Fatalf("__tkn__ path = %q, want /", c.Path)
				}
				if c.MaxAge > 0 {
					t.Fatalf("__tkn__ MaxAge = %d, want an immediate clear", c.MaxAge)
				}
			}
			if !cleared {
				t.Fatalf("response did not set __tkn__: %v", w.Result().Header.Get("Set-Cookie"))
			}
		})
	}
}
