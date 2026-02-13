package main

import (
	"nexus-banking-core/db"
	"nexus-banking-core/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	db.InitDB()
	r := gin.Default()

	r.POST("/register", handlers.RegisterUser)
	r.POST("/accounts", handlers.CreateAccount)

	r.POST("/transactions", handlers.PerformTransaction)
	r.GET("/accounts/:id", handlers.GetAccountDetails)

	r.POST("/loans", handlers.TakeLoan)
	r.POST("/loans/repay", handlers.RepayLoan)
	r.GET("/loans/:account_id", handlers.GetLoanDetails)
	r.GET("/accounts/:id/transactions", handlers.GetTransactionHistory)

	r.Run(":8080")
}
