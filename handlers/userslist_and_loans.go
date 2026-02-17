package handlers

import (
	"fmt"
	"net/http"
	"nexus-banking-core/db"
	"nexus-banking-core/models"

	"github.com/gin-gonic/gin"
)

// loan details for specific users
func GetUserLoans(c *gin.Context) {
	userID := c.Param("user_id")

	// Fetch User Details into the report struct
	var report models.UserLoanReport
	userQuery := `SELECT id AS user_id, full_name, email FROM users WHERE id = $1`

	err := db.DB.Get(&report, userQuery, userID)
	if err != nil {
		fmt.Println("DB Mapping Error:", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	//  Fetch all loans linked to any account owned by this user
	loanQuery := `
        SELECT l.* FROM loans l
        JOIN account_owners ao ON l.account_id = ao.account_id
        WHERE ao.user_id = $1`

	err = db.DB.Select(&report.Loans, loanQuery, userID)
	if err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch loans"})
		return
	}

	//  Return the report
	c.JSON(http.StatusOK, report)
}

// GetAllUsersWithLoans fetches every user and their associated loans
func GetAllUsersWithLoans(c *gin.Context) {
	var users []models.User

	// Get all users
	err := db.DB.Select(&users, "SELECT id, full_name, email FROM users ORDER BY id ASC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Get all loans with their accounts
	type LoanWithOwner struct {
		models.Loan
		UserID int64 `db:"user_id"`
	}
	var loanRows []LoanWithOwner
	query := `
        SELECT l.*, ao.user_id 

        FROM loans l
        JOIN account_owners ao ON l.account_id = ao.account_id`

	err = db.DB.Select(&loanRows, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch loans"})
		return
	}

	// Map loans to users
	loanMap := make(map[int64][]models.Loan)
	for _, lr := range loanRows {
		loanMap[lr.UserID] = append(loanMap[lr.UserID], lr.Loan)
	}

	// final response
	var response []models.UserLoanReport
	for _, u := range users {
		response = append(response, models.UserLoanReport{
			UserID:   u.ID,
			FullName: u.FullName,
			Email:    u.Email,
			Loans:    loanMap[u.ID],
		})
	}

	c.JSON(http.StatusOK, response)
}
