package handlers

import (
	"net/http"
	"nexus-banking-core/db"

	"github.com/gin-gonic/gin"
)

// TakeLoan handles the loan application and disbursements
func TakeLoan(c *gin.Context) {
	var req struct {
		AccountID       int64 `json:"account_id" binding:"required"`
		UserID          int64 `json:"user_id" binding:"required"` // Verified against account_owners
		PrincipalAmount int64 `json:"principal_amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// SECURITY: Ownership Check
	var isOwner bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM account_owners WHERE account_id = $1 AND user_id = $2)`
	err := db.DB.Get(&isOwner, checkQuery, req.AccountID, req.UserID)

	if err != nil || !isOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "User is not an authorized owner of this account"})
		return
	}

	// Start Transaction
	tx, err := db.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start transaction"})
		return
	}
	defer tx.Rollback()

	// Insert the Loan record (12% interest)
	var loanID int64
	queryLoan := `INSERT INTO loans (account_id, principal_amount, interest_rate, remaining_amount, status) 
                  VALUES ($1, $2, 12.0, $2, 'ACTIVE') RETURNING id`

	err = tx.QueryRow(queryLoan, req.AccountID, req.PrincipalAmount).Scan(&loanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create loan record"})
		return
	}

	// Update the account balance (Disburse the money)
	_, err = tx.Exec(`UPDATE accounts SET balance = balance + $1 WHERE id = $2`, req.PrincipalAmount, req.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update account balance"})
		return
	}

	// Commit the changes
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"loan_id":          loanID,
		"account_id":       req.AccountID,
		"principal_amount": req.PrincipalAmount,
		"interest_rate":    12.0,
		"status":           "ACTIVE",
		"message":          "Loan disbursed successfully",
	})
}

// RepayLoan handles paying back the loan
func RepayLoan(c *gin.Context) {
	var req struct {
		LoanID    int64 `json:"loan_id" binding:"required"`
		AccountID int64 `json:"account_id" binding:"required"`
		Amount    int64 `json:"amount" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//  SECURITY: Verify the Loan actually belongs to this specific Account
	var exists bool
	verifyQuery := `SELECT EXISTS(SELECT 1 FROM loans WHERE id = $1 AND account_id = $2)`
	err := db.DB.Get(&exists, verifyQuery, req.LoanID, req.AccountID)
	if err != nil || !exists {
		c.JSON(http.StatusForbidden, gin.H{"error": "This loan does not belong to the specified account"})
		return
	}

	tx, err := db.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}
	defer tx.Rollback()

	//  Deduct from account balance
	res, err := tx.Exec(`UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1`, req.Amount, req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient account balance"})
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment failed: Insufficient funds"})
		return
	}

	//  Reduce loan remaining amount
	_, err = tx.Exec(`UPDATE loans SET remaining_amount = remaining_amount - $1 WHERE id = $2`, req.Amount, req.LoanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update loan"})
		return
	}

	//  Update status to REPAID if fully paid
	_, err = tx.Exec(`UPDATE loans SET status = 'REPAID' WHERE id = $1 AND remaining_amount <= 0`, req.LoanID)

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Loan repayment successful"})
}

// GetLoanDetails returns active loans for an account
func GetLoanDetails(c *gin.Context) {
	accountID := c.Param("account_id")

	type LoanSummary struct {
		ID              int64   `db:"id" json:"loan_id"`
		PrincipalAmount int64   `db:"principal_amount" json:"principal_amount"`
		InterestRate    float64 `db:"interest_rate" json:"interest_rate"`
		RemainingAmount int64   `db:"remaining_amount" json:"remaining_amount"`
		Status          string  `db:"status" json:"status"`
	}

	var loans []LoanSummary
	query := `SELECT id, principal_amount, interest_rate, remaining_amount, status 
              FROM loans WHERE account_id = $1 AND status = 'ACTIVE'`

	err := db.DB.Select(&loans, query, accountID)

	if err != nil || len(loans) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "No active loans found for this account"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active_loans": loans,
	})
}
