package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"inventory_management/middleware"
)

// LoginInput is the request body for POST /login.
type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// credentials returns the single demo account this project authenticates
// against. Override via environment variables for anything beyond local
// development/grading.
func credentials() (string, string) {
	user := os.Getenv("AUTH_USERNAME")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("AUTH_PASSWORD")
	if pass == "" {
		pass = "admin123"
	}
	return user, pass
}

// Login godoc
// @Summary      Log in and receive a JWT
// @Description  Exchanges a username/password for a bearer token required by the write endpoints (POST, PUT, DELETE on /inventory).
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginInput  true  "Login credentials"
// @Success      200          {object}  map[string]string
// @Failure      400          {object}  map[string]string
// @Failure      401          {object}  map[string]string
// @Router       /login [post]
func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validUser, validPass := credentials()
	if input.Username != validUser || input.Password != validPass {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	token, err := middleware.GenerateToken(input.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
