package handlers

import (
	"fmt"
	"net/http"
	"nexus-banking-core/db"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateAccount(c *gin.Context) {
	var req struct {
		UserIDs  []int64 `json:"user_ids" binding:"required,min=1"` // Accept a list of users
		BranchID int64   `json:"branch_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	generatedAccNo := fmt.Sprintf("NEX-%d", time.Now().UnixNano())

	// Create the Account
	var accountID int64
	queryAcc := `INSERT INTO accounts (branch_id, account_number, balance, is_active) 
                 VALUES ($1, $2, 0, true) RETURNING id`
	err = tx.QueryRow(queryAcc, req.BranchID, generatedAccNo).Scan(&accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}

	//  Link all users to this account
	for i, uID := range req.UserIDs {
		role := "JOINT"
		if i == 0 {
			role = "PRIMARY"
		}

		_, err = tx.Exec(`INSERT INTO account_owners (account_id, user_id, role) VALUES ($1, $2, $3)`,
			accountID, uID, role)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "One or more User IDs are invalid"})
			return
		}
	}

	tx.Commit()
	c.JSON(http.StatusCreated, gin.H{"account_id": accountID, "account_number": generatedAccNo})
}
