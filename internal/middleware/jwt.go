package middleware

import (
	"net/http"
	"strings"

	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware(jwter *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwter.Verify(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		// Populate context with claims
		c.Set("user_id", claims.UserID)
		c.Set("provider", claims.Provider)
		c.Set("provider_id",claims.ProviderID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Next()
	}
}
