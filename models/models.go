package models

import (
	"time"
)

type Bank struct {
	ID   int64  `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type Branch struct {
	ID         int64  `json:"id" db:"id"`
	BankID     int64  `json:"bank_id" db:"bank_id"`
	IFSCCode   string `json:"ifsc_code" db:"ifsc_code"`
	BranchName string `json:"branch_name" db:"branch_name"`
}

type User struct {
	ID           int64  `json:"id" db:"id"`
	FullName     string `json:"full_name" db:"full_name"`
	Email        string `json:"email" db:"email"`
	PasswordHash string `json:"-" db:"password_hash"`
}

type Account struct {
	ID            int64     `db:"id" json:"id"`
	BranchID      int64     `db:"branch_id" json:"branch_id"`
	AccountNumber string    `db:"account_number" json:"account_number"`
	Balance       int64     `db:"balance" json:"balance"`
	IsActive      bool      `db:"is_active" json:"is_active"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type AccountOwner struct {
	AccountID int64  `db:"account_id" json:"account_id"`
	UserID    int64  `db:"user_id" json:"user_id"`
	Role      string `db:"role" json:"role"`
}

type Transaction struct {
	ID        int64     `json:"id" db:"id"`
	AccountID int64     `json:"account_id" db:"account_id"`
	Type      string    `json:"type" db:"type"`
	Amount    int64     `json:"amount" db:"amount"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Loan struct {
	ID              int64     `json:"id" db:"id"`
	AccountID       int64     `json:"account_id" db:"account_id"`
	PrincipalAmount int64     `json:"principal_amount" db:"principal_amount"`
	InterestRate    float64   `json:"interest_rate" db:"interest_rate"`
	RemainingAmount int64     `json:"remaining_amount" db:"remaining_amount"`
	Status          string    `json:"status" db:"status"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type UserLoanReport struct {
	UserID   int64  `json:"user_id" db:"user_id"` // Set this to user_id
	FullName string `json:"full_name" db:"full_name"`
	Email    string `json:"email" db:"email"`
	Loans    []Loan `json:"loans" db:"-"`
}
