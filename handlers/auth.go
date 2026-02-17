package handlers

import (
	"net/http"
	"nexus-banking-core/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// RegisterUser handles POST /register
func RegisterUser(c *gin.Context) {
	// input structure
	var req struct {
		FullName string `json:"full_name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	// Bind JSON body to the struct (checks for missing fields)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not secure password"})
		return
	}

	// Insert into Database

	var userID int64
	query := `INSERT INTO users (full_name, email, password_hash) VALUES ($1, $2, $3) RETURNING id`

	err = db.DB.QueryRow(query, req.FullName, req.Email, string(hash)).Scan(&userID)
	if err != nil {

		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully!",
		"user_id": userID,
	})
}
