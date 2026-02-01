// Package trading - Demo mode with mock transaction data for hackathon testing
package trading

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// DemoMode controls whether to use mock data
var DemoMode = true

// DemoWalletData holds the loaded JSON demo data
type DemoWalletData struct {
	Wallet struct {
		Balance  float64 `json:"balance"`
		Currency string  `json:"currency"`
		UserID   string  `json:"user_id"`
		Username string  `json:"username"`
	} `json:"wallet"`
	Savings struct {
		TotalBalance float64 `json:"total_balance"`
		Positions    []struct {
			VaultID     string  `json:"vault_id"`
			VaultName   string  `json:"vault_name"`
			Amount      float64 `json:"amount"`
			APY         float64 `json:"apy"`
			Currency    string  `json:"currency"`
			DepositedAt string  `json:"deposited_at"`
		} `json:"positions"`
	} `json:"savings"`
	VaultRates []struct {
		VaultID    string  `json:"vault_id"`
		Name       string  `json:"name"`
		APY        float64 `json:"apy"`
		Risk       string  `json:"risk"`
		MinDeposit float64 `json:"min_deposit"`
	} `json:"vault_rates"`
	Profile struct {
		UserID    string `json:"user_id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
	} `json:"profile"`
	Transactions []struct {
		ID        string  `json:"id"`
		Type      string  `json:"type"`
		Amount    float64 `json:"amount"`
		Currency  string  `json:"currency"`
		Recipient string  `json:"recipient,omitempty"`
		Sender    string  `json:"sender,omitempty"`
		Note      string  `json:"note"`
		CreatedAt string  `json:"created_at"`
		Category  string  `json:"category"`
	} `json:"transactions"`
	SpendingCategories map[string]struct {
		Budget float64 `json:"budget"`
		Color  string  `json:"color"`
		Icon   string  `json:"icon"`
	} `json:"spending_categories"`
	MonthlyBudgets struct {
		TotalIncome         float64 `json:"total_income"`
		TotalBudget         float64 `json:"total_budget"`
		SavingsGoal         float64 `json:"savings_goal"`
		EmergencyFundTarget float64 `json:"emergency_fund_target"`
	} `json:"monthly_budgets"`
}

var loadedDemoData *DemoWalletData

// LoadDemoDataFromFile loads demo data from the JSON file
func LoadDemoDataFromFile() (*DemoWalletData, error) {
	if loadedDemoData != nil {
		return loadedDemoData, nil
	}

	// Get the directory of this source file
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	jsonPath := filepath.Join(dir, "..", "demo_wallet_data.json")

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var demoData DemoWalletData
	if err := json.Unmarshal(data, &demoData); err != nil {
		return nil, err
	}

	loadedDemoData = &demoData
	return loadedDemoData, nil
}

// GetDemoDataPath returns the path to the demo data JSON file
func GetDemoDataPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "demo_wallet_data.json")
}

// GenerateMockTransactions creates realistic transaction history for demo purposes
// First tries to load from JSON file, then falls back to generated data
func GenerateMockTransactions(days int) []Transaction {
	// Try to load from JSON file first
	if demoData, err := LoadDemoDataFromFile(); err == nil && len(demoData.Transactions) > 0 {
		transactions := make([]Transaction, 0, len(demoData.Transactions))
		cutoff := time.Now().AddDate(0, 0, -days)

		for _, tx := range demoData.Transactions {
			createdAt, err := time.Parse(time.RFC3339, tx.CreatedAt)
			if err != nil {
				continue
			}

			// Only include transactions within the requested date range
			if createdAt.Before(cutoff) {
				continue
			}

			transactions = append(transactions, Transaction{
				ID:        tx.ID,
				Type:      tx.Type,
				Amount:    tx.Amount,
				Currency:  tx.Currency,
				Recipient: tx.Recipient,
				Sender:    tx.Sender,
				Note:      tx.Note,
				CreatedAt: createdAt,
			})
		}

		if len(transactions) > 0 {
			return transactions
		}
	}

	// Fallback to generated transactions if JSON file not found or empty
	rand.Seed(time.Now().UnixNano())

	transactions := []Transaction{}

	// Common recipients for variety
	recipients := []string{
		"@coffee_shop", "@grocery_mart", "@amazon", "@netflix",
		"@uber", "@spotify", "@gym_membership", "@electric_co",
		"@water_utility", "@phone_carrier", "@alice", "@bob",
		"@restaurant_xyz", "@gas_station", "@pharmacy",
	}

	// Transaction categories with typical amounts
	categories := []struct {
		recipient string
		minAmount float64
		maxAmount float64
		frequency int // times per month
		note      string
	}{
		{"@coffee_shop", 4.50, 8.00, 20, "Morning coffee"},
		{"@grocery_mart", 45.00, 150.00, 8, "Groceries"},
		{"@amazon", 15.00, 200.00, 4, "Online shopping"},
		{"@netflix", 15.99, 15.99, 1, "Monthly subscription"},
		{"@spotify", 9.99, 9.99, 1, "Music subscription"},
		{"@uber", 12.00, 45.00, 6, "Ride"},
		{"@gym_membership", 49.99, 49.99, 1, "Monthly membership"},
		{"@electric_co", 80.00, 180.00, 1, "Electricity bill"},
		{"@phone_carrier", 65.00, 85.00, 1, "Phone bill"},
		{"@restaurant_xyz", 25.00, 80.00, 5, "Dining out"},
		{"@gas_station", 35.00, 70.00, 4, "Gas"},
	}

	// Income sources
	incomes := []struct {
		sender    string
		amount    float64
		frequency int // times per month
		note      string
	}{
		{"@employer_inc", 2500.00, 2, "Salary"},
		{"@freelance_client", 500.00, 1, "Freelance work"},
		{"@cashback_rewards", 25.00, 1, "Cashback"},
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	// Generate spending transactions
	for _, cat := range categories {
		txPerPeriod := float64(cat.frequency) * float64(days) / 30.0
		numTx := int(txPerPeriod) + rand.Intn(3) - 1
		if numTx < 0 {
			numTx = 0
		}

		for i := 0; i < numTx; i++ {
			// Random date within period
			daysAgo := rand.Intn(days)
			txDate := now.AddDate(0, 0, -daysAgo)

			// Random amount within range
			amount := cat.minAmount
			if cat.maxAmount > cat.minAmount {
				amount = cat.minAmount + rand.Float64()*(cat.maxAmount-cat.minAmount)
			}

			transactions = append(transactions, Transaction{
				ID:        generateTxID(),
				Type:      "send",
				Amount:    roundTo2Decimals(amount),
				Currency:  "USD",
				Recipient: cat.recipient,
				Note:      cat.note,
				CreatedAt: txDate,
			})
		}
	}

	// Generate income transactions
	for _, inc := range incomes {
		txPerPeriod := float64(inc.frequency) * float64(days) / 30.0
		numTx := int(txPerPeriod)

		for i := 0; i < numTx; i++ {
			daysAgo := rand.Intn(days)
			txDate := now.AddDate(0, 0, -daysAgo)

			// Add some variance to income
			variance := inc.amount * 0.05 * (rand.Float64()*2 - 1)
			amount := inc.amount + variance

			transactions = append(transactions, Transaction{
				ID:        generateTxID(),
				Type:      "receive",
				Amount:    roundTo2Decimals(amount),
				Currency:  "USD",
				Sender:    inc.sender,
				Note:      inc.note,
				CreatedAt: txDate,
			})
		}
	}

	// Add some random peer-to-peer transactions
	for i := 0; i < rand.Intn(5)+2; i++ {
		daysAgo := rand.Intn(days)
		txDate := now.AddDate(0, 0, -daysAgo)

		if rand.Float32() > 0.5 {
			// Send to friend
			transactions = append(transactions, Transaction{
				ID:        generateTxID(),
				Type:      "send",
				Amount:    roundTo2Decimals(10 + rand.Float64()*90),
				Currency:  "USD",
				Recipient: recipients[rand.Intn(len(recipients))],
				Note:      "Split bill",
				CreatedAt: txDate,
			})
		} else {
			// Receive from friend
			transactions = append(transactions, Transaction{
				ID:        generateTxID(),
				Type:      "receive",
				Amount:    roundTo2Decimals(10 + rand.Float64()*90),
				Currency:  "USD",
				Sender:    recipients[rand.Intn(len(recipients))],
				Note:      "Paid back",
				CreatedAt: txDate,
			})
		}
	}

	// Add one large "anomaly" transaction
	if days >= 14 && rand.Float32() > 0.3 {
		daysAgo := rand.Intn(days/2) + 1
		transactions = append(transactions, Transaction{
			ID:        generateTxID(),
			Type:      "send",
			Amount:    roundTo2Decimals(350 + rand.Float64()*200),
			Currency:  "USD",
			Recipient: "@large_purchase",
			Note:      "Electronics/Appliance",
			CreatedAt: now.AddDate(0, 0, -daysAgo),
		})
	}

	// Add savings transactions if period is long enough
	if days >= 30 {
		// Monthly savings deposit
		for month := 0; month < days/30; month++ {
			daysAgo := month*30 + rand.Intn(5)
			if daysAgo < days {
				transactions = append(transactions, Transaction{
					ID:        generateTxID(),
					Type:      "deposit",
					Amount:    roundTo2Decimals(200 + rand.Float64()*100),
					Currency:  "USD",
					Note:      "Monthly savings",
					CreatedAt: now.AddDate(0, 0, -daysAgo),
				})
			}
		}
	}

	// Filter to only transactions after start date
	var filtered []Transaction
	for _, tx := range transactions {
		if !tx.CreatedAt.Before(startDate) {
			filtered = append(filtered, tx)
		}
	}

	return filtered
}

// GenerateMockBalance creates mock wallet balance from JSON file or fallback
func GenerateMockBalance() map[string]interface{} {
	if demoData, err := LoadDemoDataFromFile(); err == nil {
		return map[string]interface{}{
			"balances": []interface{}{
				map[string]interface{}{
					"currency": demoData.Wallet.Currency,
					"amount":   demoData.Wallet.Balance,
				},
			},
			"user_id":  demoData.Wallet.UserID,
			"username": demoData.Wallet.Username,
		}
	}

	// Fallback to random data if file not found
	return map[string]interface{}{
		"balances": []interface{}{
			map[string]interface{}{
				"currency": "USD",
				"amount":   roundTo2Decimals(1250.00 + rand.Float64()*500),
			},
		},
	}
}

// GenerateMockSavings creates mock savings positions from JSON file or fallback
func GenerateMockSavings() map[string]interface{} {
	if demoData, err := LoadDemoDataFromFile(); err == nil {
		positions := make([]interface{}, len(demoData.Savings.Positions))
		for i, pos := range demoData.Savings.Positions {
			positions[i] = map[string]interface{}{
				"vault_id":     pos.VaultID,
				"vault":        pos.VaultName,
				"amount":       pos.Amount,
				"apy":          pos.APY,
				"currency":     pos.Currency,
				"deposited_at": pos.DepositedAt,
			}
		}
		return map[string]interface{}{
			"positions":   positions,
			"total_value": demoData.Savings.TotalBalance,
		}
	}

	// Fallback to random data if file not found
	return map[string]interface{}{
		"positions": []interface{}{
			map[string]interface{}{
				"vault":    "USDC Savings",
				"amount":   roundTo2Decimals(500 + rand.Float64()*1000),
				"apy":      6.28,
				"currency": "USDC",
			},
		},
		"total_value": roundTo2Decimals(500 + rand.Float64()*1000),
	}
}

// GenerateMockVaultRates returns mock vault rates from JSON file or fallback
func GenerateMockVaultRates() map[string]interface{} {
	if demoData, err := LoadDemoDataFromFile(); err == nil {
		vaults := make([]interface{}, len(demoData.VaultRates))
		for i, vault := range demoData.VaultRates {
			vaults[i] = map[string]interface{}{
				"vault_id":    vault.VaultID,
				"name":        vault.Name,
				"apy":         vault.APY,
				"risk":        vault.Risk,
				"min_deposit": vault.MinDeposit,
			}
		}
		return map[string]interface{}{
			"vaults": vaults,
		}
	}

	// Fallback to hardcoded data if file not found
	return map[string]interface{}{
		"vaults": []interface{}{
			map[string]interface{}{
				"name":     "USDC Savings",
				"apy":      6.28,
				"currency": "USDC",
				"risk":     "low",
			},
			map[string]interface{}{
				"name":     "EURC Savings",
				"apy":      1.65,
				"currency": "EURC",
				"risk":     "low",
			},
			map[string]interface{}{
				"name":     "High Yield",
				"apy":      12.5,
				"currency": "USDC",
				"risk":     "medium",
			},
		},
	}
}

func generateTxID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := make([]byte, 16)
	for i := range id {
		id[i] = chars[rand.Intn(len(chars))]
	}
	return "tx_" + string(id)
}

func roundTo2Decimals(val float64) float64 {
	return float64(int(val*100)) / 100
}
