package handlers

import (
	"fmt"
	"net/http"
	"nexus-banking-core/db"

	"github.com/gin-gonic/gin"
)

type Customer struct {
	ID       int64  `json:"id" db:"id"`
	FullName string `json:"full_name" db:"full_name"`
	Email    string `json:"email" db:"email"`
}

// GetCustomers handles listing and searching customers
func GetCustomers(c *gin.Context) {
	searchTerm := c.Query("search")
	var customers []Customer
	var err error

	if searchTerm != "" {
		// Only selecting columns that actually exist: id, full_name, email
		query := `SELECT id, full_name, email 
                  FROM users 
                  WHERE full_name ILIKE $1 OR email ILIKE $1 
                  ORDER BY id DESC`
		err = db.DB.Select(&customers, query, "%"+searchTerm+"%")
	} else {
		query := `SELECT id, full_name, email FROM users ORDER BY id DESC`
		err = db.DB.Select(&customers, query)
	}

	if err != nil {
		fmt.Printf("SQL Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, customers)
}
