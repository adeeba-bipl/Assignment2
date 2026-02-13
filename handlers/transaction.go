package handlers

import (
	"net/http"
	"nexus-banking-core/db"
	"nexus-banking-core/models"

	"github.com/gin-gonic/gin"
)

// PerformTransaction handles DEPOSIT and WITHDRAW with Ownership Verification
func PerformTransaction(c *gin.Context) {
	var req struct {
		AccountID int64  `json:"account_id" binding:"required"`
		UserID    int64  `json:"user_id" binding:"required"` // Added to verify ownership
		Amount    int64  `json:"amount" binding:"required,gt=0"`
		Type      string `json:"type" binding:"required,oneof=DEPOSIT WITHDRAW"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input. Type must be DEPOSIT or WITHDRAW"})
		return
	}

	// SECURITY: Ownership Check
	// This ensures only people linked to the account can move money
	var isOwner bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM account_owners WHERE account_id = $1 AND user_id = $2)`
	err := db.DB.Get(&isOwner, checkQuery, req.AccountID, req.UserID)

	if err != nil || !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not an authorized owner of this account"})
		return
	}

	// Start a Database Transaction (TX)
	tx, err := db.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	defer tx.Rollback()

	// Update the Account Balance
	var query string
	if req.Type == "DEPOSIT" {
		query = `UPDATE accounts SET balance = balance + $1 WHERE id = $2 RETURNING balance`
	} else {
		// For WITHDRAW, ensure balance is enough
		query = `UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1 RETURNING balance`
	}

	var newBalance int64
	err = tx.QueryRow(query, req.Amount, req.AccountID).Scan(&newBalance)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Transaction failed: Insufficient funds or database error"})
		return
	}

	//  Log the Transaction in history
	_, err = tx.Exec(`INSERT INTO transactions (account_id, type, amount) VALUES ($1, $2, $3)`,
		req.AccountID, req.Type, req.Amount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log transaction"})
		return
	}

	// Commit the Transaction
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     req.Type + " successful",
		"new_balance": newBalance,
	})
}

// GetAccountDetails fetches account info and all associated owners
func GetAccountDetails(c *gin.Context) {
	id := c.Param("id")

	// Get the account info
	var account struct {
		ID            int64  `db:"id" json:"id"`
		AccountNumber string `db:"account_number" json:"account_number"`
		Balance       int64  `db:"balance" json:"balance"`
	}

	err := db.DB.Get(&account, "SELECT id, account_number, balance FROM accounts WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	// Fetch all owners for this account via JOIN
	type OwnerInfo struct {
		UserID   int64  `db:"user_id" json:"user_id"`
		FullName string `db:"full_name" json:"full_name"`
		Role     string `db:"role" json:"role"`
	}
	var owners []OwnerInfo

	ownerQuery := `
        SELECT ao.user_id, u.full_name, ao.role 
        FROM account_owners ao
        JOIN users u ON ao.user_id = u.id
        WHERE ao.account_id = $1`

	err = db.DB.Select(&owners, ownerQuery, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch owner details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account": account,
		"owners":  owners,
	})
}

// GetTransactionHistory returns a list of all transactions for a specific account
func GetTransactionHistory(c *gin.Context) {
	accountID := c.Param("id")

	var history []models.Transaction

	query := `SELECT id, account_id, type, amount, created_at 
              FROM transactions 
              WHERE account_id = $1 
              ORDER BY created_at DESC`

	err := db.DB.Select(&history, query, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch transaction history"})
		return
	}

	if len(history) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No transactions found for this account",
			"history": []models.Transaction{},
		})
		return
	}

	c.JSON(http.StatusOK, history)
}
