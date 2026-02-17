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

	type FlatRow struct {
		UserID   int64  `db:"user_id"`
		FullName string `db:"full_name"`
		Email    string `db:"email"`

		LoanID          *int64  `db:"loan_id"`
		PrincipalAmount *int64  `db:"principal_amount"`
		Status          *string `db:"status"`
		RemainingAmount *int64  `db:"remaining_amount"`
	}

	var rows []FlatRow

	// Single query using joins
	query := `
        SELECT u.id as user_id, u.full_name, u.email, 
               l.id as loan_id, l.principal_amount, l.status, l.remaining_amount
        FROM users u
        LEFT JOIN account_owners ao ON u.id = ao.user_id
        LEFT JOIN loans l ON ao.account_id = l.account_id
        ORDER BY u.id ASC`

	err := db.DB.Select(&rows, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	// 3. Grouping the rows
	reportMap := make(map[int64]*models.UserLoanReport)
	var finalResponse []models.UserLoanReport

	for _, row := range rows {

		if _, exists := reportMap[row.UserID]; !exists {
			userReport := &models.UserLoanReport{
				UserID:   row.UserID,
				FullName: row.FullName,
				Email:    row.Email,
				Loans:    []models.Loan{},
			}
			reportMap[row.UserID] = userReport
		}

		// If the row contains a loan
		if row.LoanID != nil {
			reportMap[row.UserID].Loans = append(reportMap[row.UserID].Loans, models.Loan{
				ID:              *row.LoanID,
				PrincipalAmount: *row.PrincipalAmount,
				Status:          *row.Status,
				RemainingAmount: *row.RemainingAmount,
			})
		}
	}

	for _, row := range rows {
		if report, exists := reportMap[row.UserID]; exists {
			finalResponse = append(finalResponse, *report)
			delete(reportMap, row.UserID)
		}
	}

	c.JSON(http.StatusOK, finalResponse)
}
