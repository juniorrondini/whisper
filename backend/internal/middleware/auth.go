package middleware

import (
	"strings"

	"whisper/backend/internal/auth"
	"whisper/backend/internal/platform/httputil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	ContextUserID    = "user_id"
	ContextCompanyID = "company_id"
	ContextRole      = "role"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" || token == header {
			httputil.Unauthorized(c)
			return
		}
		claims, err := auth.ParseAccessToken(secret, token)
		if err != nil {
			httputil.Unauthorized(c)
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextCompanyID, claims.CompanyID)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, role := range roles {
		allowed[role] = true
	}
	return func(c *gin.Context) {
		role, _ := c.Get(ContextRole)
		if !allowed[role.(string)] {
			httputil.Forbidden(c)
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) uuid.UUID {
	value, _ := c.Get(ContextUserID)
	return value.(uuid.UUID)
}

func CompanyID(c *gin.Context) uuid.UUID {
	value, _ := c.Get(ContextCompanyID)
	return value.(uuid.UUID)
}

func Role(c *gin.Context) string {
	value, _ := c.Get(ContextRole)
	return value.(string)
}
