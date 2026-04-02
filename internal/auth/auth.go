package auth

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var JWTSecret = []byte("your-secret-key")

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Register(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Save user
		id := uuid.New().String()
		_, err = db.Exec("INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)",
			id, req.Username, string(hash))
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user_id": id})
	}
}

func Login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Find user
		var id, passwordHash string
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?",
			req.Username).Scan(&id, &passwordHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Check password
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Generate JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  id,
			"username": req.Username,
			"exp":      time.Now().Add(24 * time.Hour).Unix(),
		})

		tokenString, err := token.SignedString(JWTSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString, "user_id": id})
	}
}
