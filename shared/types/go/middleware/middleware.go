// Package middleware provides shared HTTP middleware for AIAUDITOR services.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TenantContextKey is the gin context key for org ID.
const TenantContextKey = "org_id"

// UserContextKey is the gin context key for user ID.
const UserContextKey = "user_id"

// Auth validates the JWT token and sets the tenant context.
// In production this validates against the identity service.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid authorization header format",
			})
			return
		}

		// TODO: validate JWT and extract claims
		// For now, extract org_id from X-Org-ID header (dev only)
		orgID := c.GetHeader("X-Org-ID")
		if orgID != "" {
			c.Set(TenantContextKey, orgID)
		}

		c.Next()
	}
}

// RequireTenant ensures an org context is set.
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get(TenantContextKey); !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "FORBIDDEN",
				"message": "tenant context required",
			})
			return
		}
		c.Next()
	}
}

// CORS sets permissive CORS headers for development.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Org-ID")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

