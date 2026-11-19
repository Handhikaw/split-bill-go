package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS mengizinkan origin frontend yang dipisah dari backend.
//
// Kosong = middleware tidak melakukan apa-apa (same-origin / reverse proxy).
// Isi dengan daftar origin dipisah koma, contoh:
//
//	CORS_ALLOWED_ORIGINS=https://splitkita.id,https://www.splitkita.id
//
// Hanya origin persis yang cocok yang diizinkan. Jangan pakai "*" karena
// frontend mengirim X-API-Key.
func CORS(allowedOrigins string) gin.HandlerFunc {
	allowed := make(map[string]struct{})
	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" && origin != "*" {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" || len(allowed) == 0 {
			c.Next()
			return
		}

		if _, ok := allowed[origin]; !ok {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Participant-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Disposition")
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
