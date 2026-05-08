package websocket

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte(getEnvWS("JWT_SECRET", "your-secret-key"))

func getEnvWS(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func WSAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Query("token")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token query param required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		c.Set("user_id", claims["user_id"])
		c.Set("username", claims["username"])
		c.Next()
	}
}
