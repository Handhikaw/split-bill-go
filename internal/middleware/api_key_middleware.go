package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ApiKeyAuthMiddleware memverifikasi JWT yang dikirim via header X-API-Key
// (dipakai frontend sebagai pengganti Authorization: Bearer).
// Tanpa key / key invalid -> lanjut sebagai guest (tanpa 401).
// Kalau valid -> set "auth_user_id" (int64) + "auth_role" (string) di context,
// key SAMA dengan OptionalAuthMiddleware agar handler tidak peduli sumbernya.
func ApiKeyAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtSecret == "" {
			c.Next()
			return
		}

		tokenStr := c.GetHeader("X-API-Key")
		if tokenStr == "" {
			c.Next()
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.Next()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Next()
			return
		}

		if rawID, ok := claims["user_id"]; ok {
			var userID int64
			switch v := rawID.(type) {
			case float64:
				userID = int64(v)
			case int64:
				userID = v
			case int:
				userID = int64(v)
			}
			if userID > 0 {
				c.Set("auth_user_id", userID)
			}
		}
		if role, ok := claims["role"].(string); ok && role != "" {
			c.Set("auth_role", role)
		}

		c.Next()
	}
}
