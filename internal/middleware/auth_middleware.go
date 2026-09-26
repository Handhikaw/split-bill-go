package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func ParticipantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		pIDHeader := c.GetHeader("X-Participant-ID")
		if pIDHeader != "" {
			pID, err := strconv.ParseInt(pIDHeader, 10, 64)
			if err == nil {
				// Simpan ke context kalau datanya valid
				c.Set("participant_id", pID)
			}
		}

		// Tetap lanjut ke Handler, baik ada ID maupun tidak
		c.Next()
	}
}

// OptionalAuthMiddleware memverifikasi Bearer JWT bila ada.
// Tanpa token / token invalid -> lanjut sebagai guest (tanpa 401).
// Kalau valid -> set "auth_user_id" (int64) + "auth_role" (string) di context.
func OptionalAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if jwtSecret == "" {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.Next()
			return
		}
		tokenStr := strings.TrimSpace(parts[1])
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

// AuthUserID mengambil user_id dari context bila ada (hasil OptionalAuthMiddleware).
func AuthUserID(c *gin.Context) (*int64, bool) {
	v, ok := c.Get("auth_user_id")
	if !ok {
		return nil, false
	}
	id, ok := v.(int64)
	if !ok || id <= 0 {
		return nil, false
	}
	return &id, true
}
