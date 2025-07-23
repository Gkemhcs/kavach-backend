package middleware

import (
	"strings"

	"github.com/Gkemhcs/kavach-backend/internal/auth/jwt"
	apperrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware returns a Gin middleware that authenticates requests using JWT tokens.
// It validates the Authorization header, verifies the token, and populates context with user claims.
func JWTAuthMiddleware(jwter *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			utils.RespondError(c, apperrors.ErrInvalidToken.Status, apperrors.ErrInvalidToken.Code, apperrors.ErrInvalidToken.Message)
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwter.Verify(tokenStr)
		if err != nil {
			if err == apperrors.ErrExpiredToken {
				utils.RespondError(c, apperrors.ErrExpiredToken.Status, apperrors.ErrExpiredToken.Code, apperrors.ErrExpiredToken.Message)
				return
			}
			if err == apperrors.ErrInvalidToken {
				utils.RespondError(c, apperrors.ErrInvalidToken.Status, apperrors.ErrInvalidToken.Code, apperrors.ErrInvalidToken.Message)
				return
			}
			utils.RespondError(c, apperrors.ErrInvalidToken.Status, apperrors.ErrInvalidToken.Code, apperrors.ErrInvalidToken.Message)
			return
		}
		// Populate context with claims for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("provider", claims.Provider)
		c.Set("provider_id", claims.ProviderID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Next()
	}
}
