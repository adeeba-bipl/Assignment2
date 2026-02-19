package handlers

import (
	"fmt"
	"net/http"
	"nexus-banking-core/db"

	"github.com/gin-gonic/gin"
)

// GetAllAccountsWithTransactions lists every account and its history
func GetAllAccountsWithTransactions(c *gin.Context) {

	type AccountTxRow struct {
		AccountID     int64   `db:"account_id"`
		AccountNumber string  `db:"account_number"`
		Balance       int64   `db:"balance"`
		TxID          *int64  `db:"tx_id"`
		TxType        *string `db:"tx_type"`
		TxAmount      *int64  `db:"tx_amount"`
	}

	var rows []AccountTxRow

	//  Query using left join

	query := `
        SELECT a.id as account_id, a.account_number, a.balance,
               t.id as tx_id, t.type as tx_type, t.amount as tx_amount
        FROM accounts a
        LEFT JOIN transactions t ON a.id = t.account_id
        ORDER BY a.id DESC, t.id DESC`

	err := db.DB.Select(&rows, query)
	if err != nil {
		fmt.Printf("SQL Error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	type AccountReport struct {
		AccountID     int64         `json:"account_id"`
		AccountNumber string        `json:"account_number"`
		Balance       int64         `json:"balance"`
		Transactions  []interface{} `json:"transactions"`
	}

	reportMap := make(map[int64]*AccountReport)
	var finalResponse []*AccountReport

	for _, row := range rows {

		if _, exists := reportMap[row.AccountID]; !exists {
			acc := &AccountReport{
				AccountID:     row.AccountID,
				AccountNumber: row.AccountNumber,
				Balance:       row.Balance,
				Transactions:  []interface{}{},
			}
			reportMap[row.AccountID] = acc
			// Append to finalResponse
			finalResponse = append(finalResponse, acc)
		}

		// add transaction
		if row.TxID != nil {
			reportMap[row.AccountID].Transactions = append(reportMap[row.AccountID].Transactions, gin.H{
				"id":     *row.TxID,
				"type":   *row.TxType,
				"amount": *row.TxAmount,
			})
		}
	}

	// Return the data
	c.JSON(http.StatusOK, finalResponse)
}
