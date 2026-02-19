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
		UserID          int64 `json:"user_id" binding:"required"`
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

	//  Insert the Loan record
	var loanID int64
	queryLoan := `INSERT INTO loans (account_id, principal_amount, interest_rate, remaining_amount, status) 
                  VALUES ($1, $2, 12.0, $2, 'ACTIVE') RETURNING id`

	err = tx.QueryRow(queryLoan, req.AccountID, req.PrincipalAmount).Scan(&loanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create loan record"})
		return
	}

	// Record the disbursement in the transactions table
	txQuery := `INSERT INTO transactions (account_id, type, amount) VALUES ($1, 'LOAN_DISBURSEMENT', $2)`
	_, err = tx.Exec(txQuery, req.AccountID, req.PrincipalAmount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not log disbursement transaction"})
		return
	}

	// Update the account balance
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
		"message":          "Loan disbursed and added to transaction history",
	})
}

// RepayLoan handles loan repayment
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

	tx, err := db.DB.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}
	defer tx.Rollback()

	// SAFETY CHECK: Get current remaining amount
	var remaining int64
	err = tx.Get(&remaining, "SELECT remaining_amount FROM loans WHERE id = $1 FOR UPDATE", req.LoanID) //FOR UPDATE to lock the row during repayment
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Loan not found"})
		return
	}

	if req.Amount > remaining {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Repayment amount exceeds remaining loan balance"})
		return
	}

	// DEDUCT from Account Balance
	res, err := tx.Exec(`UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1`, req.Amount, req.AccountID)
	if err != nil || func() bool { r, _ := res.RowsAffected(); return r == 0 }() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds to repay loan"})
		return
	}

	// SHOW in Transaction History
	_, err = tx.Exec(`INSERT INTO transactions (account_id, type, amount) VALUES ($1, 'LOAN_REPAYMENT', $2)`,
		req.AccountID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log transaction"})
		return
	}

	// REDUCE the Loan Balance
	_, err = tx.Exec(`UPDATE loans SET remaining_amount = remaining_amount - $1 WHERE id = $2`, req.Amount, req.LoanID)

	// Mark as REPAID if balance hits zero
	tx.Exec(`UPDATE loans SET status = 'REPAID' WHERE id = $1 AND remaining_amount <= 0`, req.LoanID)

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Repayment processed and logged in history"})
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
