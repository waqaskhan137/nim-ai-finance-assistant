// Package trading - Hackathon-compatible insight tools
// These tools analyze Liminal wallet data to provide non-obvious financial insights
package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// ============================================================================
// INSIGHT TYPES - Structured outputs for Claude to explain
// ============================================================================

// SpendingInsight represents analyzed spending patterns
type SpendingInsight struct {
	Period             string             `json:"period"`
	TotalSpent         float64            `json:"total_spent"`
	TotalReceived      float64            `json:"total_received"`
	NetFlow            float64            `json:"net_flow"`
	AvgTransactionSize float64            `json:"avg_transaction_size"`
	TransactionCount   int                `json:"transaction_count"`
	SpendingVelocity   string             `json:"spending_velocity"` // "low", "moderate", "high"
	TopRecipients      []RecipientSummary `json:"top_recipients"`
	Anomalies          []Anomaly          `json:"anomalies"`
	Trends             TrendAnalysis      `json:"trends"`
	Recommendations    []string           `json:"recommendations"`
}

// RecipientSummary shows spending by recipient
type RecipientSummary struct {
	Recipient   string  `json:"recipient"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
}

// Anomaly represents unusual transaction patterns
type Anomaly struct {
	Type        string  `json:"type"` // "large_transaction", "unusual_frequency", "new_recipient"
	Description string  `json:"description"`
	Severity    string  `json:"severity"` // "info", "warning", "alert"
	Amount      float64 `json:"amount,omitempty"`
	Date        string  `json:"date,omitempty"`
}

// TrendAnalysis shows spending trends over time
type TrendAnalysis struct {
	Direction        string  `json:"direction"` // "increasing", "decreasing", "stable"
	PercentChange    float64 `json:"percent_change"`
	ProjectedMonthly float64 `json:"projected_monthly"`
	Comparison       string  `json:"comparison"` // Human-readable comparison
}

// SavingsOpportunity represents potential savings optimization
type SavingsOpportunity struct {
	CurrentAPY       float64 `json:"current_apy"`
	BestAvailableAPY float64 `json:"best_available_apy"`
	PotentialGain    float64 `json:"potential_gain_yearly"`
	IdleCash         float64 `json:"idle_cash"`
	SuggestedDeposit float64 `json:"suggested_deposit"`
	Recommendation   string  `json:"recommendation"`
	RiskLevel        string  `json:"risk_level"`
}

// TradingReadiness assesses if user is ready for trading
type TradingReadiness struct {
	Score             int      `json:"score"` // 0-100
	CanTrade          bool     `json:"can_trade"`
	AvailableFunds    float64  `json:"available_funds"`
	RecommendedBudget float64  `json:"recommended_budget"`
	RiskAssessment    string   `json:"risk_assessment"`
	Warnings          []string `json:"warnings"`
	Suggestions       []string `json:"suggestions"`
}

// FinancialHealthScore provides overall financial health assessment
type FinancialHealthScore struct {
	OverallScore      int              `json:"overall_score"` // 0-100
	Category          string           `json:"category"`      // "excellent", "good", "fair", "needs_attention"
	Components        HealthComponents `json:"components"`
	Insights          []string         `json:"insights"`
	ActionItems       []ActionItem     `json:"action_items"`
	ComparisonToPeers string           `json:"comparison_to_peers"`
}

// HealthComponents breaks down the health score
type HealthComponents struct {
	SavingsRate     int `json:"savings_rate"`     // 0-100
	SpendingControl int `json:"spending_control"` // 0-100
	EmergencyFund   int `json:"emergency_fund"`   // 0-100
	GrowthPotential int `json:"growth_potential"` // 0-100
}

// ActionItem is a specific recommended action
type ActionItem struct {
	Priority  string `json:"priority"` // "high", "medium", "low"
	Action    string `json:"action"`
	Impact    string `json:"impact"`
	TimeToAct string `json:"time_to_act"`
}

// ============================================================================
// INSIGHT TOOLS - Hackathon Compatible
// ============================================================================

// CreateAnalyzeSpendingPatternsTool creates a tool to analyze spending patterns
// This provides NON-OBVIOUS insights by analyzing transaction history
func CreateAnalyzeSpendingPatternsTool(executor core.ToolExecutor) core.Tool {
	return tools.New("analyze_spending_patterns").
		Description("Analyze user's spending patterns to identify trends, anomalies, and provide actionable insights. Returns non-obvious findings like spending velocity changes, unusual transactions, and personalized recommendations.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"days":            tools.IntegerProperty("Number of days to analyze (default: 30, max: 90)"),
			"include_savings": tools.BooleanProperty("Include savings transactions in analysis"),
		})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				Days           int  `json:"days"`
				IncludeSavings bool `json:"include_savings"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				return &core.ToolResult{Success: false, Error: "Invalid input parameters"}, nil
			}

			if params.Days <= 0 {
				params.Days = 30
			}
			if params.Days > 90 {
				params.Days = 90
			}

			var transactions []Transaction

			// Use demo data if enabled, otherwise fetch from Liminal API
			if DemoMode {
				transactions = GenerateMockTransactions(params.Days)
			} else {
				// Fetch transactions from Liminal API
				txRequest := map[string]interface{}{"limit": 200}
				txRequestJSON, _ := json.Marshal(txRequest)

				txResponse, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_transactions",
					Input:     txRequestJSON,
					RequestID: toolParams.RequestID,
				})

				if err != nil || !txResponse.Success {
					errMsg := "Failed to fetch transactions"
					if err != nil {
						errMsg = err.Error()
					} else if txResponse.Error != "" {
						errMsg = txResponse.Error
					}
					return &core.ToolResult{Success: false, Error: errMsg}, nil
				}

				// Parse transactions
				var txData struct {
					Transactions []Transaction `json:"transactions"`
				}
				if err := json.Unmarshal(txResponse.Data, &txData); err != nil {
					return &core.ToolResult{Success: false, Error: "Failed to parse transaction data"}, nil
				}
				transactions = txData.Transactions
			}

			// Handle empty data gracefully
			if len(transactions) == 0 {
				return &core.ToolResult{
					Success: true,
					Data: SpendingInsight{
						Period:           fmt.Sprintf("Last %d days", params.Days),
						Recommendations:  []string{"Start making transactions to build spending history for analysis"},
						SpendingVelocity: "none",
					},
				}, nil
			}

			// Add demo mode indicator
			txData := struct {
				Transactions []Transaction `json:"transactions"`
			}{Transactions: transactions}

			// Analyze the transactions
			insight := analyzeTransactionsForInsights(txData.Transactions, params.Days, params.IncludeSavings)

			return &core.ToolResult{
				Success: true,
				Data:    insight,
			}, nil
		}).
		Build()
}

// CreateSavingsOptimizerTool finds opportunities to optimize savings
func CreateSavingsOptimizerTool(executor core.ToolExecutor) core.Tool {
	return tools.New("optimize_savings").
		Description("Analyze current savings positions against available vault rates to find optimization opportunities. Identifies idle cash that could be earning interest and calculates potential gains.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"risk_tolerance": tools.StringEnumProperty("User's risk tolerance for savings", "low", "medium", "high"),
		})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				RiskTolerance string `json:"risk_tolerance"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				params.RiskTolerance = "low"
			}
			if params.RiskTolerance == "" {
				params.RiskTolerance = "low"
			}

			var balanceData, savingsData, ratesData map[string]interface{}

			if DemoMode {
				// Use mock data for demo
				balanceData = GenerateMockBalance()
				savingsData = GenerateMockSavings()
				ratesData = GenerateMockVaultRates()
			} else {
				// Fetch balance, savings, and vault rates from Liminal API

				// Get wallet balance
				balanceResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_balance",
					Input:     []byte("{}"),
					RequestID: toolParams.RequestID,
				})
				if err != nil || !balanceResp.Success {
					return &core.ToolResult{Success: false, Error: "Failed to fetch balance"}, nil
				}

				// Get savings balance
				savingsResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_savings_balance",
					Input:     []byte("{}"),
					RequestID: toolParams.RequestID,
				})
				if err != nil || !savingsResp.Success {
					return &core.ToolResult{Success: false, Error: "Failed to fetch savings"}, nil
				}

				// Get vault rates
				ratesResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_vault_rates",
					Input:     []byte("{}"),
					RequestID: toolParams.RequestID,
				})
				if err != nil || !ratesResp.Success {
					return &core.ToolResult{Success: false, Error: "Failed to fetch vault rates"}, nil
				}

				// Parse responses
				json.Unmarshal(balanceResp.Data, &balanceData)
				json.Unmarshal(savingsResp.Data, &savingsData)
				json.Unmarshal(ratesResp.Data, &ratesData)
			}

			// Calculate optimization opportunity
			opportunity := calculateSavingsOpportunity(balanceData, savingsData, ratesData, params.RiskTolerance)

			return &core.ToolResult{
				Success: true,
				Data:    opportunity,
			}, nil
		}).
		Build()
}

// CreateTradingReadinessAssessmentTool assesses if user should start trading
func CreateTradingReadinessAssessmentTool(executor core.ToolExecutor, portfolio *Portfolio) core.Tool {
	return tools.New("assess_trading_readiness").
		Description("Assess whether the user is financially ready to start trading. Analyzes wallet balance, spending patterns, emergency fund status, and provides a readiness score with personalized recommendations.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var balanceData map[string]interface{}
			var txData struct{ Transactions []Transaction }
			var savingsData map[string]interface{}

			if DemoMode {
				// Use mock data for demo
				balanceData = GenerateMockBalance()
				txData.Transactions = GenerateMockTransactions(30)
				savingsData = GenerateMockSavings()
			} else {
				// Fetch balance
				balanceResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_balance",
					Input:     []byte("{}"),
					RequestID: toolParams.RequestID,
				})
				if err != nil || !balanceResp.Success {
					return &core.ToolResult{Success: false, Error: "Failed to fetch balance"}, nil
				}

				// Fetch transactions for spending analysis
				txRequest := map[string]interface{}{"limit": 100}
				txRequestJSON, _ := json.Marshal(txRequest)
				txResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_transactions",
					Input:     txRequestJSON,
					RequestID: toolParams.RequestID,
				})

				// Fetch savings
				savingsResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "get_savings_balance",
					Input:     []byte("{}"),
					RequestID: toolParams.RequestID,
				})

				// Parse data
				json.Unmarshal(balanceResp.Data, &balanceData)
				if txResp != nil && txResp.Success {
					json.Unmarshal(txResp.Data, &txData)
				}
				if savingsResp != nil && savingsResp.Success {
					json.Unmarshal(savingsResp.Data, &savingsData)
				}
			}

			// Calculate readiness
			readiness := assessTradingReadiness(balanceData, txData.Transactions, savingsData, portfolio)

			return &core.ToolResult{
				Success: true,
				Data:    readiness,
			}, nil
		}).
		Build()
}

// CreateFinancialHealthScoreTool calculates overall financial health
func CreateFinancialHealthScoreTool(executor core.ToolExecutor) core.Tool {
	return tools.New("calculate_financial_health").
		Description("Calculate a comprehensive financial health score (0-100) based on spending habits, savings rate, emergency fund status, and growth potential. Provides actionable insights and comparisons.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var balanceData, savingsData, ratesData map[string]interface{}
			var txData struct{ Transactions []Transaction }

			if DemoMode {
				// Use mock data for demo
				balanceData = GenerateMockBalance()
				savingsData = GenerateMockSavings()
				ratesData = GenerateMockVaultRates()
				txData.Transactions = GenerateMockTransactions(30)
			} else {
				// Fetch all relevant data
				balanceResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_balance", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				savingsResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_savings_balance", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				txReq, _ := json.Marshal(map[string]interface{}{"limit": 100})
				txResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_transactions", Input: txReq, RequestID: toolParams.RequestID,
				})
				ratesResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_vault_rates", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})

				// Parse all data
				if balanceResp != nil && balanceResp.Success {
					json.Unmarshal(balanceResp.Data, &balanceData)
				}
				if savingsResp != nil && savingsResp.Success {
					json.Unmarshal(savingsResp.Data, &savingsData)
				}
				if txResp != nil && txResp.Success {
					json.Unmarshal(txResp.Data, &txData)
				}
				if ratesResp != nil && ratesResp.Success {
					json.Unmarshal(ratesResp.Data, &ratesData)
				}
			}

			// Calculate health score
			healthScore := calculateFinancialHealth(balanceData, savingsData, txData.Transactions, ratesData)

			return &core.ToolResult{
				Success: true,
				Data:    healthScore,
			}, nil
		}).
		Build()
}

// CreateSmartBudgetRecommendationTool provides AI-driven budget recommendations
func CreateSmartBudgetRecommendationTool(executor core.ToolExecutor) core.Tool {
	return tools.New("get_smart_budget").
		Description("Generate personalized budget recommendations based on income, spending patterns, and financial goals. Uses transaction history to create realistic, achievable budget targets.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"goal": tools.StringEnumProperty("Primary financial goal", "save_more", "reduce_spending", "grow_wealth", "emergency_fund"),
		})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				Goal string `json:"goal"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				params.Goal = "save_more"
			}
			if params.Goal == "" {
				params.Goal = "save_more"
			}

			var txData struct{ Transactions []Transaction }

			if DemoMode {
				// Use mock data for demo
				txData.Transactions = GenerateMockTransactions(60)
			} else {
				// Fetch transaction history
				txReq, _ := json.Marshal(map[string]interface{}{"limit": 200})
				txResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_transactions", Input: txReq, RequestID: toolParams.RequestID,
				})
				if err != nil || !txResp.Success {
					return &core.ToolResult{Success: false, Error: "Failed to fetch transaction history"}, nil
				}
				json.Unmarshal(txResp.Data, &txData)
			}

			// Generate budget recommendations
			budget := generateSmartBudget(txData.Transactions, params.Goal)

			return &core.ToolResult{
				Success: true,
				Data:    budget,
			}, nil
		}).
		Build()
}

// ============================================================================
// ANALYSIS HELPER FUNCTIONS
// ============================================================================

// Transaction represents a Liminal transaction
type Transaction struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "send", "receive", "deposit", "withdraw"
	Amount    float64   `json:"amount"`
	Currency  string    `json:"currency"`
	Recipient string    `json:"recipient,omitempty"`
	Sender    string    `json:"sender,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func analyzeTransactionsForInsights(transactions []Transaction, days int, includeSavings bool) SpendingInsight {
	cutoff := time.Now().AddDate(0, 0, -days)

	var totalSpent, totalReceived float64
	var sendCount, receiveCount int
	recipientMap := make(map[string]float64)
	recipientCountMap := make(map[string]int)
	var amounts []float64
	var anomalies []Anomaly

	// First pass: calculate totals and identify patterns
	for _, tx := range transactions {
		if tx.CreatedAt.Before(cutoff) {
			continue
		}

		if !includeSavings && (tx.Type == "deposit" || tx.Type == "withdraw") {
			continue
		}

		switch tx.Type {
		case "send":
			totalSpent += tx.Amount
			sendCount++
			amounts = append(amounts, tx.Amount)
			if tx.Recipient != "" {
				recipientMap[tx.Recipient] += tx.Amount
				recipientCountMap[tx.Recipient]++
			}
		case "receive":
			totalReceived += tx.Amount
			receiveCount++
		case "deposit":
			totalSpent += tx.Amount // Moving to savings
			sendCount++
		case "withdraw":
			totalReceived += tx.Amount // From savings
			receiveCount++
		}
	}

	txCount := sendCount + receiveCount
	avgTxSize := 0.0
	if sendCount > 0 {
		avgTxSize = totalSpent / float64(sendCount)
	}

	// Calculate standard deviation for anomaly detection
	if len(amounts) > 3 {
		mean := totalSpent / float64(len(amounts))
		var variance float64
		for _, a := range amounts {
			variance += (a - mean) * (a - mean)
		}
		stdDev := math.Sqrt(variance / float64(len(amounts)))

		// Identify anomalies (transactions > 2 std dev from mean)
		for _, tx := range transactions {
			if tx.Type == "send" && tx.Amount > mean+2*stdDev {
				anomalies = append(anomalies, Anomaly{
					Type:        "large_transaction",
					Description: fmt.Sprintf("Unusually large payment of $%.2f (%.1fx your average)", tx.Amount, tx.Amount/mean),
					Severity:    "warning",
					Amount:      tx.Amount,
					Date:        tx.CreatedAt.Format("2006-01-02"),
				})
			}
		}
	}

	// Build top recipients list
	var topRecipients []RecipientSummary
	for recipient, amount := range recipientMap {
		pct := 0.0
		if totalSpent > 0 {
			pct = (amount / totalSpent) * 100
		}
		topRecipients = append(topRecipients, RecipientSummary{
			Recipient:   recipient,
			TotalAmount: amount,
			Count:       recipientCountMap[recipient],
			Percentage:  pct,
		})
	}
	// Sort by amount descending
	sort.Slice(topRecipients, func(i, j int) bool {
		return topRecipients[i].TotalAmount > topRecipients[j].TotalAmount
	})
	if len(topRecipients) > 5 {
		topRecipients = topRecipients[:5]
	}

	// Calculate spending velocity
	velocity := "low"
	txPerWeek := float64(txCount) / float64(days) * 7
	if txPerWeek >= 10 {
		velocity = "high"
	} else if txPerWeek >= 4 {
		velocity = "moderate"
	}

	// Calculate trends (compare first half vs second half)
	halfDays := days / 2
	halfCutoff := time.Now().AddDate(0, 0, -halfDays)
	var firstHalfSpent, secondHalfSpent float64
	for _, tx := range transactions {
		if tx.CreatedAt.Before(cutoff) {
			continue
		}
		if tx.Type == "send" {
			if tx.CreatedAt.Before(halfCutoff) {
				firstHalfSpent += tx.Amount
			} else {
				secondHalfSpent += tx.Amount
			}
		}
	}

	trendDirection := "stable"
	pctChange := 0.0
	if firstHalfSpent > 0 {
		pctChange = ((secondHalfSpent - firstHalfSpent) / firstHalfSpent) * 100
		if pctChange > 15 {
			trendDirection = "increasing"
		} else if pctChange < -15 {
			trendDirection = "decreasing"
		}
	}

	projectedMonthly := (totalSpent / float64(days)) * 30

	// Generate recommendations
	var recommendations []string
	if velocity == "high" {
		recommendations = append(recommendations, "Consider batching smaller transactions to reduce frequency")
	}
	if len(anomalies) > 0 {
		recommendations = append(recommendations, "Review large transactions to ensure they align with your budget")
	}
	if totalSpent > totalReceived*1.2 {
		recommendations = append(recommendations, "Spending exceeds income - consider reducing non-essential expenses")
	}
	if len(topRecipients) > 0 && topRecipients[0].Percentage > 50 {
		recommendations = append(recommendations, fmt.Sprintf("%.0f%% of spending goes to %s - diversify if unintentional", topRecipients[0].Percentage, topRecipients[0].Recipient))
	}
	if trendDirection == "increasing" {
		recommendations = append(recommendations, fmt.Sprintf("Spending increased %.0f%% recently - monitor closely", pctChange))
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Your spending patterns look healthy - keep it up!")
	}

	comparison := fmt.Sprintf("Spending %.0f%% %s compared to previous period", math.Abs(pctChange), trendDirection)

	return SpendingInsight{
		Period:             fmt.Sprintf("Last %d days", days),
		TotalSpent:         totalSpent,
		TotalReceived:      totalReceived,
		NetFlow:            totalReceived - totalSpent,
		AvgTransactionSize: avgTxSize,
		TransactionCount:   txCount,
		SpendingVelocity:   velocity,
		TopRecipients:      topRecipients,
		Anomalies:          anomalies,
		Trends: TrendAnalysis{
			Direction:        trendDirection,
			PercentChange:    pctChange,
			ProjectedMonthly: projectedMonthly,
			Comparison:       comparison,
		},
		Recommendations: recommendations,
	}
}

func calculateSavingsOpportunity(balance, savings, rates map[string]interface{}, riskTolerance string) SavingsOpportunity {
	// Extract wallet balance (idle cash)
	idleCash := 0.0
	if balances, ok := balance["balances"].([]interface{}); ok {
		for _, b := range balances {
			if bMap, ok := b.(map[string]interface{}); ok {
				if currency, _ := bMap["currency"].(string); currency == "USD" {
					if amount, ok := bMap["amount"].(float64); ok {
						idleCash = amount
					}
				}
			}
		}
	}

	// Extract current savings APY
	currentAPY := 0.0
	if positions, ok := savings["positions"].([]interface{}); ok {
		for _, p := range positions {
			if pMap, ok := p.(map[string]interface{}); ok {
				if apy, ok := pMap["apy"].(float64); ok {
					if apy > currentAPY {
						currentAPY = apy
					}
				}
			}
		}
	}

	// Find best available rate based on risk tolerance
	bestAPY := 0.0
	if vaults, ok := rates["vaults"].([]interface{}); ok {
		for _, v := range vaults {
			if vMap, ok := v.(map[string]interface{}); ok {
				apy, _ := vMap["apy"].(float64)
				risk, _ := vMap["risk"].(string)

				// Filter by risk tolerance
				if riskTolerance == "low" && risk != "low" {
					continue
				}
				if riskTolerance == "medium" && risk == "high" {
					continue
				}

				if apy > bestAPY {
					bestAPY = apy
				}
			}
		}
	}

	// Default rates if not available
	if bestAPY == 0 {
		bestAPY = 5.0 // Default 5% APY
	}

	// Calculate potential gain
	suggestedDeposit := idleCash * 0.8 // Suggest depositing 80% of idle cash
	potentialGain := suggestedDeposit * (bestAPY / 100)

	recommendation := ""
	if idleCash > 100 && bestAPY > currentAPY {
		recommendation = fmt.Sprintf("You have $%.2f idle in your wallet earning 0%%. Depositing $%.2f into savings could earn you $%.2f/year at %.1f%% APY.",
			idleCash, suggestedDeposit, potentialGain, bestAPY)
	} else if currentAPY > 0 {
		recommendation = "Your funds are already earning competitive rates. Consider increasing deposits as income allows."
	} else {
		recommendation = "Start building your savings to take advantage of competitive APY rates."
	}

	return SavingsOpportunity{
		CurrentAPY:       currentAPY,
		BestAvailableAPY: bestAPY,
		PotentialGain:    potentialGain,
		IdleCash:         idleCash,
		SuggestedDeposit: suggestedDeposit,
		Recommendation:   recommendation,
		RiskLevel:        riskTolerance,
	}
}

func assessTradingReadiness(balance map[string]interface{}, transactions []Transaction, savings map[string]interface{}, portfolio *Portfolio) TradingReadiness {
	// Calculate available funds
	availableFunds := 0.0
	if balances, ok := balance["balances"].([]interface{}); ok {
		for _, b := range balances {
			if bMap, ok := b.(map[string]interface{}); ok {
				if currency, _ := bMap["currency"].(string); currency == "USD" {
					if amount, ok := bMap["amount"].(float64); ok {
						availableFunds = amount
					}
				}
			}
		}
	}

	// Calculate savings balance for emergency fund check
	savingsBalance := 0.0
	if positions, ok := savings["positions"].([]interface{}); ok {
		for _, p := range positions {
			if pMap, ok := p.(map[string]interface{}); ok {
				if amount, ok := pMap["amount"].(float64); ok {
					savingsBalance += amount
				}
			}
		}
	}

	// Calculate monthly spending from transactions
	monthlySpending := 0.0
	cutoff := time.Now().AddDate(0, -1, 0)
	for _, tx := range transactions {
		if tx.CreatedAt.After(cutoff) && tx.Type == "send" {
			monthlySpending += tx.Amount
		}
	}

	// Calculate readiness score (0-100)
	score := 0
	var warnings, suggestions []string

	// Emergency fund check (3 months of spending)
	emergencyFundTarget := monthlySpending * 3
	if savingsBalance >= emergencyFundTarget {
		score += 30
	} else if savingsBalance >= monthlySpending {
		score += 15
		warnings = append(warnings, "Emergency fund is below recommended 3 months of expenses")
	} else {
		warnings = append(warnings, "Build an emergency fund before trading")
	}

	// Available funds check
	recommendedBudget := availableFunds * 0.1 // Only trade with 10% of available
	if availableFunds >= 100 {
		score += 25
	} else if availableFunds >= 50 {
		score += 15
		warnings = append(warnings, "Limited funds available for trading")
	} else {
		warnings = append(warnings, "Insufficient funds for meaningful trading")
	}

	// Income vs spending check
	if len(transactions) > 0 {
		totalReceived := 0.0
		totalSpent := 0.0
		for _, tx := range transactions {
			if tx.CreatedAt.After(cutoff) {
				if tx.Type == "receive" {
					totalReceived += tx.Amount
				} else if tx.Type == "send" {
					totalSpent += tx.Amount
				}
			}
		}

		if totalReceived > totalSpent*1.2 {
			score += 25
		} else if totalReceived > totalSpent {
			score += 15
			suggestions = append(suggestions, "Increase income or reduce spending before allocating to trading")
		} else {
			warnings = append(warnings, "Spending exceeds income - not recommended to trade")
		}
	}

	// Existing portfolio check
	if portfolio != nil {
		status := portfolio.GetStatus()
		if canTrade, ok := status["can_trade"].(bool); ok && canTrade {
			score += 20
		}
	} else {
		score += 10 // Neutral
	}

	// Generate suggestions
	if score >= 70 {
		suggestions = append(suggestions, fmt.Sprintf("You're ready to start! Recommended budget: $%.2f (10%% of available funds)", recommendedBudget))
	} else if score >= 50 {
		suggestions = append(suggestions, "Consider starting with a small amount ($10-20) to learn")
	} else {
		suggestions = append(suggestions, "Focus on building savings and reducing spending first")
	}

	// Risk assessment
	riskAssessment := "low"
	if score < 50 {
		riskAssessment = "high"
	} else if score < 70 {
		riskAssessment = "medium"
	}

	canTrade := score >= 50 && availableFunds >= 20

	return TradingReadiness{
		Score:             score,
		CanTrade:          canTrade,
		AvailableFunds:    availableFunds,
		RecommendedBudget: recommendedBudget,
		RiskAssessment:    riskAssessment,
		Warnings:          warnings,
		Suggestions:       suggestions,
	}
}

func calculateFinancialHealth(balance, savings map[string]interface{}, transactions []Transaction, rates map[string]interface{}) FinancialHealthScore {
	var insights []string
	var actionItems []ActionItem

	// Extract data
	walletBalance := 0.0
	savingsBalance := 0.0
	if balances, ok := balance["balances"].([]interface{}); ok {
		for _, b := range balances {
			if bMap, ok := b.(map[string]interface{}); ok {
				if currency, _ := bMap["currency"].(string); currency == "USD" {
					if amount, ok := bMap["amount"].(float64); ok {
						walletBalance = amount
					}
				}
			}
		}
	}
	if positions, ok := savings["positions"].([]interface{}); ok {
		for _, p := range positions {
			if pMap, ok := p.(map[string]interface{}); ok {
				if amount, ok := pMap["amount"].(float64); ok {
					savingsBalance += amount
				}
			}
		}
	}

	totalAssets := walletBalance + savingsBalance
	_ = totalAssets // Used for future enhancements

	// Calculate monthly income/spending
	monthlyIncome := 0.0
	monthlySpending := 0.0
	cutoff := time.Now().AddDate(0, -1, 0)
	for _, tx := range transactions {
		if tx.CreatedAt.After(cutoff) {
			if tx.Type == "receive" {
				monthlyIncome += tx.Amount
			} else if tx.Type == "send" {
				monthlySpending += tx.Amount
			}
		}
	}

	// Component scores (0-100 each)
	components := HealthComponents{}

	// Savings rate score
	if monthlyIncome > 0 {
		savingsRate := (monthlyIncome - monthlySpending) / monthlyIncome
		components.SavingsRate = int(math.Min(savingsRate*200, 100)) // 50% savings = 100
		if savingsRate > 0.2 {
			insights = append(insights, fmt.Sprintf("Excellent savings rate of %.0f%% - you're saving more than average!", savingsRate*100))
		} else if savingsRate < 0 {
			insights = append(insights, "You're spending more than you earn - this needs attention")
			actionItems = append(actionItems, ActionItem{Priority: "high", Action: "Create a budget to track spending", Impact: "Stop negative cash flow", TimeToAct: "This week"})
		}
	}

	// Spending control score
	if monthlyIncome > 0 && monthlySpending > 0 {
		spendingRatio := monthlySpending / monthlyIncome
		components.SpendingControl = int(math.Max(0, 100-(spendingRatio*100)))
	} else {
		components.SpendingControl = 50 // Neutral
	}

	// Emergency fund score
	emergencyTarget := monthlySpending * 3
	if emergencyTarget > 0 {
		emergencyRatio := savingsBalance / emergencyTarget
		components.EmergencyFund = int(math.Min(emergencyRatio*100, 100))
		if emergencyRatio < 0.5 {
			actionItems = append(actionItems, ActionItem{Priority: "high", Action: "Build emergency fund to 3 months expenses", Impact: "Financial security", TimeToAct: "3-6 months"})
		}
	} else {
		components.EmergencyFund = 50
	}

	// Growth potential score
	if savingsBalance > 0 {
		components.GrowthPotential = 70 // Has savings working
		insights = append(insights, fmt.Sprintf("Your savings of $%.2f are growing - consider optimizing APY", savingsBalance))
	} else if walletBalance > 100 {
		components.GrowthPotential = 40
		actionItems = append(actionItems, ActionItem{Priority: "medium", Action: "Move idle funds to savings to earn interest", Impact: "Passive income", TimeToAct: "Today"})
	} else {
		components.GrowthPotential = 20
	}

	// Calculate overall score (weighted average)
	overallScore := (components.SavingsRate*30 + components.SpendingControl*25 + components.EmergencyFund*25 + components.GrowthPotential*20) / 100

	// Determine category
	category := "needs_attention"
	if overallScore >= 80 {
		category = "excellent"
	} else if overallScore >= 60 {
		category = "good"
	} else if overallScore >= 40 {
		category = "fair"
	}

	// Peer comparison (mock - in production would use real data)
	comparison := "Your financial health is "
	if overallScore >= 70 {
		comparison += "above average compared to similar users"
	} else if overallScore >= 50 {
		comparison += "on par with similar users"
	} else {
		comparison += "below average - but with small changes you can improve quickly"
	}

	return FinancialHealthScore{
		OverallScore:      overallScore,
		Category:          category,
		Components:        components,
		Insights:          insights,
		ActionItems:       actionItems,
		ComparisonToPeers: comparison,
	}
}

func generateSmartBudget(transactions []Transaction, goal string) map[string]interface{} {
	// Analyze spending by category (simulated - in production would use transaction categories)
	monthlyIncome := 0.0
	monthlySpending := 0.0
	cutoff := time.Now().AddDate(0, -1, 0)

	for _, tx := range transactions {
		if tx.CreatedAt.After(cutoff) {
			if tx.Type == "receive" {
				monthlyIncome += tx.Amount
			} else if tx.Type == "send" {
				monthlySpending += tx.Amount
			}
		}
	}

	// Calculate budget based on goal
	savingsTarget := 0.0
	spendingLimit := monthlySpending

	switch goal {
	case "save_more":
		savingsTarget = monthlyIncome * 0.20
		spendingLimit = monthlyIncome - savingsTarget
	case "reduce_spending":
		spendingLimit = monthlySpending * 0.85 // 15% reduction
		savingsTarget = monthlyIncome - spendingLimit
	case "grow_wealth":
		savingsTarget = monthlyIncome * 0.30
		spendingLimit = monthlyIncome - savingsTarget
	case "emergency_fund":
		savingsTarget = monthlyIncome * 0.25
		spendingLimit = monthlyIncome - savingsTarget
	}

	recommendations := []string{}
	if monthlySpending > monthlyIncome {
		recommendations = append(recommendations, fmt.Sprintf("You're overspending by $%.2f/month - prioritize reducing expenses", monthlySpending-monthlyIncome))
	}
	if savingsTarget > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Set up automatic transfers of $%.2f/month to savings", savingsTarget))
	}

	return map[string]interface{}{
		"goal":                       goal,
		"current_monthly_income":     monthlyIncome,
		"current_monthly_spending":   monthlySpending,
		"recommended_spending_limit": spendingLimit,
		"recommended_savings":        savingsTarget,
		"savings_rate":               fmt.Sprintf("%.0f%%", (savingsTarget/monthlyIncome)*100),
		"spending_reduction_needed":  math.Max(0, monthlySpending-spendingLimit),
		"recommendations":            recommendations,
		"achievable":                 monthlyIncome >= spendingLimit,
	}
}

// ============================================================================
// BUDGET PLANNER - AI-Driven Insights & Wealth Forecasting
// ============================================================================

// BudgetPlan represents a comprehensive budget plan with forecasting
type BudgetPlan struct {
	// Current Financial Snapshot
	CurrentSnapshot FinancialSnapshot `json:"current_snapshot"`

	// Spending Trackers (Daily, Weekly, Monthly)
	SpendingTrackers SpendingTrackers `json:"spending_trackers"`

	// Recent Transactions
	RecentTransactions []TransactionDetail `json:"recent_transactions"`

	// Spending Breakdown by Category
	SpendingByCategory []CategorySpending `json:"spending_by_category"`

	// Budget Recommendations
	RecommendedBudget RecommendedBudget `json:"recommended_budget"`

	// Wealth Forecast
	WealthForecast WealthForecast `json:"wealth_forecast"`

	// Optimization Opportunities
	Optimizations []OptimizationOpportunity `json:"optimizations"`

	// Action Plan
	ActionPlan []BudgetAction `json:"action_plan"`

	// Overall Assessment
	BudgetScore    int    `json:"budget_score"` // 0-100
	BudgetGrade    string `json:"budget_grade"` // A, B, C, D, F
	SummaryInsight string `json:"summary_insight"`
}

// SpendingTrackers contains daily, weekly, and monthly spending data
type SpendingTrackers struct {
	Daily   DailyTracker   `json:"daily"`
	Weekly  WeeklyTracker  `json:"weekly"`
	Monthly MonthlyTracker `json:"monthly"`
}

// DailyTracker tracks spending by day
type DailyTracker struct {
	Today           DaySpending   `json:"today"`
	Yesterday       DaySpending   `json:"yesterday"`
	Last7Days       []DaySpending `json:"last_7_days"`
	DailyAverage    float64       `json:"daily_average"`
	DailyBudget     float64       `json:"daily_budget"`
	TodayVsBudget   string        `json:"today_vs_budget"` // "under", "on_track", "over"
	HighestSpendDay string        `json:"highest_spend_day"`
}

// DaySpending represents spending for a single day
type DaySpending struct {
	Date             string              `json:"date"`
	DayOfWeek        string              `json:"day_of_week"`
	TotalSpent       float64             `json:"total_spent"`
	TotalReceived    float64             `json:"total_received"`
	NetFlow          float64             `json:"net_flow"`
	TransactionCount int                 `json:"transaction_count"`
	TopCategory      string              `json:"top_category"`
	Transactions     []TransactionDetail `json:"transactions,omitempty"`
}

// WeeklyTracker tracks spending by week
type WeeklyTracker struct {
	CurrentWeek        WeekSpending   `json:"current_week"`
	LastWeek           WeekSpending   `json:"last_week"`
	Last4Weeks         []WeekSpending `json:"last_4_weeks"`
	WeeklyAverage      float64        `json:"weekly_average"`
	WeeklyBudget       float64        `json:"weekly_budget"`
	CurrentVsBudget    string         `json:"current_vs_budget"`
	WeekOverWeekChange float64        `json:"week_over_week_change_percent"`
	BestWeek           string         `json:"best_week"`
	WorstWeek          string         `json:"worst_week"`
}

// WeekSpending represents spending for a week
type WeekSpending struct {
	WeekStart         string             `json:"week_start"`
	WeekEnd           string             `json:"week_end"`
	WeekNumber        int                `json:"week_number"`
	TotalSpent        float64            `json:"total_spent"`
	TotalReceived     float64            `json:"total_received"`
	NetFlow           float64            `json:"net_flow"`
	TransactionCount  int                `json:"transaction_count"`
	DailyBreakdown    []DaySpending      `json:"daily_breakdown,omitempty"`
	CategoryBreakdown map[string]float64 `json:"category_breakdown"`
	TopCategory       string             `json:"top_category"`
}

// MonthlyTracker tracks spending by month
type MonthlyTracker struct {
	CurrentMonth         MonthSpending   `json:"current_month"`
	LastMonth            MonthSpending   `json:"last_month"`
	Last6Months          []MonthSpending `json:"last_6_months"`
	MonthlyAverage       float64         `json:"monthly_average"`
	MonthlyBudget        float64         `json:"monthly_budget"`
	CurrentVsBudget      string          `json:"current_vs_budget"`
	MonthOverMonthChange float64         `json:"month_over_month_change_percent"`
	ProjectedEndOfMonth  float64         `json:"projected_end_of_month"`
	DaysRemaining        int             `json:"days_remaining"`
	DailyBudgetRemaining float64         `json:"daily_budget_remaining"`
}

// MonthSpending represents spending for a month
type MonthSpending struct {
	Month             string             `json:"month"`       // "January 2026"
	MonthShort        string             `json:"month_short"` // "Jan"
	Year              int                `json:"year"`
	TotalSpent        float64            `json:"total_spent"`
	TotalReceived     float64            `json:"total_received"`
	NetFlow           float64            `json:"net_flow"`
	SavingsRate       float64            `json:"savings_rate_percent"`
	TransactionCount  int                `json:"transaction_count"`
	CategoryBreakdown map[string]float64 `json:"category_breakdown"`
	TopCategories     []CategoryAmount   `json:"top_categories"`
	WeeklyBreakdown   []WeekSpending     `json:"weekly_breakdown,omitempty"`
}

// CategoryAmount for simple category-amount pairs
type CategoryAmount struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Percent  float64 `json:"percent"`
}

// TransactionDetail represents a single transaction with context
type TransactionDetail struct {
	ID          string  `json:"id"`
	Date        string  `json:"date"`
	Time        string  `json:"time"`
	Type        string  `json:"type"` // "send", "receive", "deposit", "withdraw"
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Recipient   string  `json:"recipient,omitempty"`
	Sender      string  `json:"sender,omitempty"`
	Note        string  `json:"note,omitempty"`
	DayOfWeek   string  `json:"day_of_week"`
	IsLarge     bool    `json:"is_large"`     // Larger than average
	IsRecurring bool    `json:"is_recurring"` // Appears to be recurring
}

// FinancialSnapshot captures current financial state
type FinancialSnapshot struct {
	TotalAssets        float64 `json:"total_assets"`
	WalletBalance      float64 `json:"wallet_balance"`
	SavingsBalance     float64 `json:"savings_balance"`
	MonthlyIncome      float64 `json:"monthly_income"`
	MonthlyExpenses    float64 `json:"monthly_expenses"`
	NetMonthlyCashFlow float64 `json:"net_monthly_cash_flow"`
	SavingsRate        float64 `json:"savings_rate_percent"`
	RunwayMonths       float64 `json:"runway_months"` // How long savings last without income
}

// CategorySpending represents spending in a category
type CategorySpending struct {
	Category         string  `json:"category"`
	Amount           float64 `json:"amount"`
	Percentage       float64 `json:"percentage"`
	TransactionCount int     `json:"transaction_count"`
	Trend            string  `json:"trend"` // "increasing", "decreasing", "stable"
	IsOptimizable    bool    `json:"is_optimizable"`
	SuggestedLimit   float64 `json:"suggested_limit"`
}

// RecommendedBudget provides target allocations
type RecommendedBudget struct {
	NeedsPercent     float64            `json:"needs_percent"`   // 50% rule
	WantsPercent     float64            `json:"wants_percent"`   // 30% rule
	SavingsPercent   float64            `json:"savings_percent"` // 20% rule
	NeedsAmount      float64            `json:"needs_amount"`
	WantsAmount      float64            `json:"wants_amount"`
	SavingsAmount    float64            `json:"savings_amount"`
	CategoryLimits   map[string]float64 `json:"category_limits"`
	AdjustmentNeeded float64            `json:"adjustment_needed"`
	BudgetMethod     string             `json:"budget_method"` // "50/30/20", "zero-based", "envelope"
}

// WealthForecast projects future wealth
type WealthForecast struct {
	CurrentNetWorth     float64           `json:"current_net_worth"`
	ProjectedNetWorth   []ProjectedWealth `json:"projected_net_worth"`
	TimeToGoal          string            `json:"time_to_goal"`
	ProjectedRetirement float64           `json:"projected_retirement_age"`
	WealthGrowthRate    float64           `json:"wealth_growth_rate_percent"`
	Scenarios           ForecastScenarios `json:"scenarios"`
}

// ProjectedWealth represents wealth at a point in time
type ProjectedWealth struct {
	Period   string  `json:"period"` // "1 month", "6 months", "1 year", etc.
	Amount   float64 `json:"amount"`
	Interest float64 `json:"interest_earned"`
}

// ForecastScenarios shows different wealth trajectories
type ForecastScenarios struct {
	Pessimistic ScenarioDetail `json:"pessimistic"`
	Expected    ScenarioDetail `json:"expected"`
	Optimistic  ScenarioDetail `json:"optimistic"`
}

// ScenarioDetail describes a forecast scenario
type ScenarioDetail struct {
	OneYear     float64 `json:"one_year"`
	FiveYears   float64 `json:"five_years"`
	TenYears    float64 `json:"ten_years"`
	Assumptions string  `json:"assumptions"`
}

// OptimizationOpportunity represents a way to save money
type OptimizationOpportunity struct {
	Category          string  `json:"category"`
	CurrentSpending   float64 `json:"current_spending"`
	OptimizedSpending float64 `json:"optimized_spending"`
	PotentialSavings  float64 `json:"potential_savings_monthly"`
	AnnualImpact      float64 `json:"annual_impact"`
	Difficulty        string  `json:"difficulty"` // "easy", "medium", "hard"
	Suggestion        string  `json:"suggestion"`
	ImpactOnLifestyle string  `json:"impact_on_lifestyle"` // "minimal", "moderate", "significant"
}

// BudgetAction is a specific action to take
type BudgetAction struct {
	Priority     int     `json:"priority"` // 1 = highest
	Action       string  `json:"action"`
	Category     string  `json:"category"`
	ImpactAmount float64 `json:"impact_amount"`
	Timeframe    string  `json:"timeframe"`
	Difficulty   string  `json:"difficulty"`
}

// CreateBudgetPlannerTool creates the comprehensive budget planner tool
func CreateBudgetPlannerTool(executor core.ToolExecutor) core.Tool {
	return tools.New("create_budget_plan").
		Description("Create a comprehensive AI-driven budget plan with spending analysis, optimization recommendations, and wealth forecasting. Analyzes your income, expenses, and savings to provide personalized insights and project your future wealth.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"goal": tools.StringEnumProperty(
				"Your primary financial goal",
				"build_wealth", "save_for_emergency", "reduce_debt", "retire_early", "general_savings",
			),
			"target_amount": tools.NumberProperty("Target savings amount (optional, for goal tracking)"),
			"risk_tolerance": tools.StringEnumProperty(
				"Investment risk tolerance for wealth projections",
				"conservative", "moderate", "aggressive",
			),
			"forecast_years": tools.IntegerProperty("Number of years to forecast (default: 5, max: 30)"),
		})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				Goal          string  `json:"goal"`
				TargetAmount  float64 `json:"target_amount"`
				RiskTolerance string  `json:"risk_tolerance"`
				ForecastYears int     `json:"forecast_years"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				params.Goal = "general_savings"
			}

			// Set defaults
			if params.Goal == "" {
				params.Goal = "general_savings"
			}
			if params.RiskTolerance == "" {
				params.RiskTolerance = "moderate"
			}
			if params.ForecastYears <= 0 {
				params.ForecastYears = 5
			}
			if params.ForecastYears > 30 {
				params.ForecastYears = 30
			}

			// Fetch financial data
			var balanceData, savingsData, ratesData map[string]interface{}
			var transactions []Transaction

			if DemoMode {
				balanceData = GenerateMockBalance()
				savingsData = GenerateMockSavings()
				ratesData = GenerateMockVaultRates()
				transactions = GenerateMockTransactions(90) // 3 months of data
			} else {
				// Fetch from Liminal API
				balanceResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_balance", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				savingsResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_savings_balance", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				ratesResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_vault_rates", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				txReq, _ := json.Marshal(map[string]interface{}{"limit": 500})
				txResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_transactions", Input: txReq, RequestID: toolParams.RequestID,
				})

				if balanceResp != nil && balanceResp.Success {
					json.Unmarshal(balanceResp.Data, &balanceData)
				}
				if savingsResp != nil && savingsResp.Success {
					json.Unmarshal(savingsResp.Data, &savingsData)
				}
				if ratesResp != nil && ratesResp.Success {
					json.Unmarshal(ratesResp.Data, &ratesData)
				}
				if txResp != nil && txResp.Success {
					var txData struct{ Transactions []Transaction }
					json.Unmarshal(txResp.Data, &txData)
					transactions = txData.Transactions
				}
			}

			// Generate the budget plan
			budgetPlan := generateBudgetPlan(
				balanceData, savingsData, ratesData, transactions,
				params.Goal, params.TargetAmount, params.RiskTolerance, params.ForecastYears,
			)

			return &core.ToolResult{
				Success: true,
				Data:    budgetPlan,
			}, nil
		}).
		Build()
}

// generateBudgetPlan creates a comprehensive budget plan
func generateBudgetPlan(
	balance, savings, rates map[string]interface{},
	transactions []Transaction,
	goal string, targetAmount float64, riskTolerance string, forecastYears int,
) BudgetPlan {

	// Extract current balances
	walletBalance := extractBalance(balance)
	savingsBalance := extractSavingsBalance(savings)
	savingsAPY := extractSavingsAPY(savings, rates)

	// Analyze transactions
	monthlyIncome, monthlyExpenses, categorySpending := analyzeTransactionsForBudget(transactions)

	// Calculate current snapshot
	netCashFlow := monthlyIncome - monthlyExpenses
	savingsRate := 0.0
	if monthlyIncome > 0 {
		savingsRate = (netCashFlow / monthlyIncome) * 100
	}
	runwayMonths := 0.0
	if monthlyExpenses > 0 {
		runwayMonths = (walletBalance + savingsBalance) / monthlyExpenses
	}

	snapshot := FinancialSnapshot{
		TotalAssets:        walletBalance + savingsBalance,
		WalletBalance:      walletBalance,
		SavingsBalance:     savingsBalance,
		MonthlyIncome:      monthlyIncome,
		MonthlyExpenses:    monthlyExpenses,
		NetMonthlyCashFlow: netCashFlow,
		SavingsRate:        savingsRate,
		RunwayMonths:       runwayMonths,
	}

	// Build spending trackers (daily, weekly, monthly)
	spendingTrackers := buildSpendingTrackers(transactions, monthlyExpenses)

	// Build recent transactions list
	recentTransactions := buildRecentTransactions(transactions, monthlyExpenses)

	// Generate recommended budget (50/30/20 rule adapted to user)
	recommendedBudget := generateRecommendedBudget(monthlyIncome, monthlyExpenses, categorySpending, goal)

	// Generate wealth forecast
	wealthForecast := generateWealthForecast(
		walletBalance+savingsBalance, netCashFlow, savingsAPY,
		riskTolerance, forecastYears, targetAmount,
	)

	// Find optimization opportunities
	optimizations := findOptimizationOpportunities(categorySpending, monthlyIncome, goal)

	// Create action plan
	actionPlan := createActionPlan(snapshot, categorySpending, optimizations, goal)

	// Calculate budget score
	budgetScore, budgetGrade := calculateBudgetScore(snapshot, categorySpending, goal)

	// Generate summary insight
	summaryInsight := generateSummaryInsight(snapshot, wealthForecast, budgetScore, goal)

	return BudgetPlan{
		CurrentSnapshot:    snapshot,
		SpendingTrackers:   spendingTrackers,
		RecentTransactions: recentTransactions,
		SpendingByCategory: categorySpending,
		RecommendedBudget:  recommendedBudget,
		WealthForecast:     wealthForecast,
		Optimizations:      optimizations,
		ActionPlan:         actionPlan,
		BudgetScore:        budgetScore,
		BudgetGrade:        budgetGrade,
		SummaryInsight:     summaryInsight,
	}
}

// Helper functions for budget planner

func extractBalance(balance map[string]interface{}) float64 {
	if balances, ok := balance["balances"].([]interface{}); ok {
		for _, b := range balances {
			if bMap, ok := b.(map[string]interface{}); ok {
				if currency, _ := bMap["currency"].(string); currency == "USD" {
					if amount, ok := bMap["amount"].(float64); ok {
						return amount
					}
				}
			}
		}
	}
	return 0
}

func extractSavingsBalance(savings map[string]interface{}) float64 {
	total := 0.0
	if positions, ok := savings["positions"].([]interface{}); ok {
		for _, p := range positions {
			if pMap, ok := p.(map[string]interface{}); ok {
				if amount, ok := pMap["amount"].(float64); ok {
					total += amount
				}
			}
		}
	}
	return total
}

func extractSavingsAPY(savings, rates map[string]interface{}) float64 {
	// Try to get from savings first
	if positions, ok := savings["positions"].([]interface{}); ok {
		for _, p := range positions {
			if pMap, ok := p.(map[string]interface{}); ok {
				if apy, ok := pMap["apy"].(float64); ok && apy > 0 {
					return apy
				}
			}
		}
	}
	// Fall back to best available rate
	bestAPY := 4.5 // Default
	if vaults, ok := rates["vaults"].([]interface{}); ok {
		for _, v := range vaults {
			if vMap, ok := v.(map[string]interface{}); ok {
				if apy, ok := vMap["apy"].(float64); ok && apy > bestAPY {
					bestAPY = apy
				}
			}
		}
	}
	return bestAPY
}

func analyzeTransactionsForBudget(transactions []Transaction) (float64, float64, []CategorySpending) {
	cutoff := time.Now().AddDate(0, -1, 0) // Last month

	var totalIncome, totalExpenses float64
	categoryMap := make(map[string]struct {
		amount float64
		count  int
	})

	for _, tx := range transactions {
		if tx.CreatedAt.Before(cutoff) {
			continue
		}

		switch tx.Type {
		case "receive":
			totalIncome += tx.Amount
		case "send":
			totalExpenses += tx.Amount
			// Categorize based on recipient/note (simplified categorization)
			category := categorizeTransaction(tx)
			entry := categoryMap[category]
			entry.amount += tx.Amount
			entry.count++
			categoryMap[category] = entry
		}
	}

	// Convert to slice and calculate percentages
	var categorySpending []CategorySpending
	for cat, data := range categoryMap {
		pct := 0.0
		if totalExpenses > 0 {
			pct = (data.amount / totalExpenses) * 100
		}

		// Determine if optimizable and suggest limits
		isOptimizable := cat != "essentials" && cat != "bills" && cat != "transfers"
		suggestedLimit := data.amount
		if isOptimizable {
			suggestedLimit = data.amount * 0.85 // Suggest 15% reduction for non-essentials
		}

		categorySpending = append(categorySpending, CategorySpending{
			Category:         cat,
			Amount:           data.amount,
			Percentage:       pct,
			TransactionCount: data.count,
			Trend:            "stable", // Would need historical data to calculate
			IsOptimizable:    isOptimizable,
			SuggestedLimit:   suggestedLimit,
		})
	}

	// Sort by amount descending
	sort.Slice(categorySpending, func(i, j int) bool {
		return categorySpending[i].Amount > categorySpending[j].Amount
	})

	return totalIncome, totalExpenses, categorySpending
}

func categorizeTransaction(tx Transaction) string {
	// Simple categorization based on recipient name or note
	// In production, this would use ML or more sophisticated matching
	recipient := strings.ToLower(tx.Recipient)
	note := strings.ToLower(tx.Note)
	combined := recipient + " " + note

	// Check for common categories
	categories := map[string][]string{
		"food_dining":    {"restaurant", "food", "cafe", "coffee", "lunch", "dinner", "uber eats", "doordash", "grubhub"},
		"shopping":       {"amazon", "store", "shop", "retail", "walmart", "target", "ebay"},
		"entertainment":  {"netflix", "spotify", "hulu", "gaming", "movie", "concert", "subscription"},
		"transportation": {"uber", "lyft", "gas", "fuel", "parking", "transit"},
		"bills":          {"electric", "water", "internet", "phone", "insurance", "rent", "mortgage"},
		"health":         {"pharmacy", "doctor", "hospital", "gym", "fitness"},
		"transfers":      {"transfer", "venmo", "zelle", "paypal", "cash app"},
		"investments":    {"invest", "stock", "crypto", "trading"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				return category
			}
		}
	}

	return "other"
}

func generateRecommendedBudget(monthlyIncome, monthlyExpenses float64, categorySpending []CategorySpending, goal string) RecommendedBudget {
	// Adjust 50/30/20 based on goal
	var needsPct, wantsPct, savingsPct float64

	switch goal {
	case "build_wealth", "retire_early":
		needsPct, wantsPct, savingsPct = 50, 20, 30
	case "save_for_emergency":
		needsPct, wantsPct, savingsPct = 50, 25, 25
	case "reduce_debt":
		needsPct, wantsPct, savingsPct = 50, 20, 30 // 30% goes to debt
	default:
		needsPct, wantsPct, savingsPct = 50, 30, 20
	}

	needsAmount := monthlyIncome * (needsPct / 100)
	wantsAmount := monthlyIncome * (wantsPct / 100)
	savingsAmount := monthlyIncome * (savingsPct / 100)

	// Calculate category limits
	categoryLimits := make(map[string]float64)
	for _, cat := range categorySpending {
		if cat.Category == "bills" || cat.Category == "essentials" {
			categoryLimits[cat.Category] = cat.Amount // Keep essentials as-is
		} else {
			// Reduce discretionary by goal-appropriate amount
			reduction := 0.15
			if goal == "build_wealth" || goal == "retire_early" {
				reduction = 0.25
			}
			categoryLimits[cat.Category] = cat.Amount * (1 - reduction)
		}
	}

	adjustmentNeeded := monthlyExpenses - (needsAmount + wantsAmount)
	if adjustmentNeeded < 0 {
		adjustmentNeeded = 0
	}

	return RecommendedBudget{
		NeedsPercent:     needsPct,
		WantsPercent:     wantsPct,
		SavingsPercent:   savingsPct,
		NeedsAmount:      needsAmount,
		WantsAmount:      wantsAmount,
		SavingsAmount:    savingsAmount,
		CategoryLimits:   categoryLimits,
		AdjustmentNeeded: adjustmentNeeded,
		BudgetMethod:     "50/30/20 (Adjusted)",
	}
}

func generateWealthForecast(currentWealth, monthlySavings, apy float64, riskTolerance string, years int, targetAmount float64) WealthForecast {
	// Determine expected returns based on risk tolerance
	var expectedReturn, pessimisticReturn, optimisticReturn float64

	switch riskTolerance {
	case "conservative":
		expectedReturn, pessimisticReturn, optimisticReturn = 4.0, 2.0, 6.0
	case "aggressive":
		expectedReturn, pessimisticReturn, optimisticReturn = 8.0, 3.0, 12.0
	default: // moderate
		expectedReturn, pessimisticReturn, optimisticReturn = 6.0, 3.0, 9.0
	}

	// Use savings APY if higher than expected return
	if apy > expectedReturn {
		expectedReturn = apy
	}

	// Calculate projected wealth at various points
	projections := []ProjectedWealth{}
	periods := []struct {
		label  string
		months int
	}{
		{"1 month", 1},
		{"3 months", 3},
		{"6 months", 6},
		{"1 year", 12},
		{"2 years", 24},
		{"5 years", 60},
	}

	for _, p := range periods {
		if p.months/12 <= years {
			wealth, interest := projectWealth(currentWealth, monthlySavings, expectedReturn, p.months)
			projections = append(projections, ProjectedWealth{
				Period:   p.label,
				Amount:   wealth,
				Interest: interest,
			})
		}
	}

	// Calculate scenarios
	pessimistic1Y, _ := projectWealth(currentWealth, monthlySavings*0.8, pessimisticReturn, 12)
	pessimistic5Y, _ := projectWealth(currentWealth, monthlySavings*0.8, pessimisticReturn, 60)
	pessimistic10Y, _ := projectWealth(currentWealth, monthlySavings*0.8, pessimisticReturn, 120)

	expected1Y, _ := projectWealth(currentWealth, monthlySavings, expectedReturn, 12)
	expected5Y, _ := projectWealth(currentWealth, monthlySavings, expectedReturn, 60)
	expected10Y, _ := projectWealth(currentWealth, monthlySavings, expectedReturn, 120)

	optimistic1Y, _ := projectWealth(currentWealth, monthlySavings*1.2, optimisticReturn, 12)
	optimistic5Y, _ := projectWealth(currentWealth, monthlySavings*1.2, optimisticReturn, 60)
	optimistic10Y, _ := projectWealth(currentWealth, monthlySavings*1.2, optimisticReturn, 120)

	scenarios := ForecastScenarios{
		Pessimistic: ScenarioDetail{
			OneYear:     pessimistic1Y,
			FiveYears:   pessimistic5Y,
			TenYears:    pessimistic10Y,
			Assumptions: fmt.Sprintf("%.1f%% return, 20%% less savings", pessimisticReturn),
		},
		Expected: ScenarioDetail{
			OneYear:     expected1Y,
			FiveYears:   expected5Y,
			TenYears:    expected10Y,
			Assumptions: fmt.Sprintf("%.1f%% return, current savings rate", expectedReturn),
		},
		Optimistic: ScenarioDetail{
			OneYear:     optimistic1Y,
			FiveYears:   optimistic5Y,
			TenYears:    optimistic10Y,
			Assumptions: fmt.Sprintf("%.1f%% return, 20%% more savings", optimisticReturn),
		},
	}

	// Calculate time to goal
	timeToGoal := "Not set"
	if targetAmount > 0 && monthlySavings > 0 {
		months := calculateTimeToGoal(currentWealth, monthlySavings, expectedReturn, targetAmount)
		if months < 12 {
			timeToGoal = fmt.Sprintf("%d months", months)
		} else {
			timeToGoal = fmt.Sprintf("%.1f years", float64(months)/12)
		}
	}

	// Simplified retirement projection (assumes $1M needed, 4% withdrawal)
	projectedRetirement := 65.0
	if monthlySavings > 0 {
		monthsToRetirement := calculateTimeToGoal(currentWealth, monthlySavings, expectedReturn, 1000000)
		currentAge := 30.0 // Assumed, would get from user profile
		projectedRetirement = currentAge + float64(monthsToRetirement)/12
		if projectedRetirement < 30 {
			projectedRetirement = 30
		}
		if projectedRetirement > 100 {
			projectedRetirement = 100
		}
	}

	return WealthForecast{
		CurrentNetWorth:     currentWealth,
		ProjectedNetWorth:   projections,
		TimeToGoal:          timeToGoal,
		ProjectedRetirement: projectedRetirement,
		WealthGrowthRate:    expectedReturn,
		Scenarios:           scenarios,
	}
}

func projectWealth(principal, monthlyContribution, annualReturn float64, months int) (float64, float64) {
	monthlyRate := annualReturn / 100 / 12
	total := principal

	for i := 0; i < months; i++ {
		total = total*(1+monthlyRate) + monthlyContribution
	}

	totalContributions := principal + (monthlyContribution * float64(months))
	interestEarned := total - totalContributions

	return total, interestEarned
}

func calculateTimeToGoal(principal, monthlyContribution, annualReturn, goal float64) int {
	if monthlyContribution <= 0 {
		return 9999
	}

	monthlyRate := annualReturn / 100 / 12
	total := principal
	months := 0

	for total < goal && months < 600 { // Max 50 years
		total = total*(1+monthlyRate) + monthlyContribution
		months++
	}

	return months
}

func findOptimizationOpportunities(categorySpending []CategorySpending, monthlyIncome float64, goal string) []OptimizationOpportunity {
	var opportunities []OptimizationOpportunity

	for _, cat := range categorySpending {
		if !cat.IsOptimizable || cat.Amount < 10 {
			continue
		}

		// Calculate potential savings
		reductionPct := 0.15 // Default 15% reduction
		difficulty := "medium"
		impactOnLifestyle := "moderate"

		switch cat.Category {
		case "entertainment", "shopping":
			reductionPct = 0.25
			difficulty = "easy"
			impactOnLifestyle = "minimal"
		case "food_dining":
			reductionPct = 0.20
			difficulty = "medium"
			impactOnLifestyle = "moderate"
		case "transportation":
			reductionPct = 0.10
			difficulty = "hard"
			impactOnLifestyle = "moderate"
		}

		if goal == "build_wealth" || goal == "retire_early" {
			reductionPct *= 1.2 // More aggressive for wealth building
		}

		potentialSavings := cat.Amount * reductionPct
		optimizedSpending := cat.Amount - potentialSavings

		suggestion := generateOptimizationSuggestion(cat.Category, potentialSavings)

		opportunities = append(opportunities, OptimizationOpportunity{
			Category:          cat.Category,
			CurrentSpending:   cat.Amount,
			OptimizedSpending: optimizedSpending,
			PotentialSavings:  potentialSavings,
			AnnualImpact:      potentialSavings * 12,
			Difficulty:        difficulty,
			Suggestion:        suggestion,
			ImpactOnLifestyle: impactOnLifestyle,
		})
	}

	// Sort by potential savings descending
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].PotentialSavings > opportunities[j].PotentialSavings
	})

	// Return top 5 opportunities
	if len(opportunities) > 5 {
		opportunities = opportunities[:5]
	}

	return opportunities
}

func generateOptimizationSuggestion(category string, savings float64) string {
	suggestions := map[string]string{
		"food_dining":    "Try meal prepping 2-3 days per week and limit dining out to weekends",
		"entertainment":  "Review subscriptions and cancel unused services; look for free alternatives",
		"shopping":       "Implement a 48-hour rule before non-essential purchases",
		"transportation": "Consider carpooling, public transit, or combining trips",
		"other":          "Track these expenses more closely to identify specific reduction opportunities",
	}

	if suggestion, ok := suggestions[category]; ok {
		return fmt.Sprintf("%s - potential savings: $%.2f/month", suggestion, savings)
	}
	return fmt.Sprintf("Review spending in this category - potential savings: $%.2f/month", savings)
}

func createActionPlan(snapshot FinancialSnapshot, categorySpending []CategorySpending, optimizations []OptimizationOpportunity, goal string) []BudgetAction {
	var actions []BudgetAction
	priority := 1

	// Priority 1: Fix negative cash flow
	if snapshot.NetMonthlyCashFlow < 0 {
		actions = append(actions, BudgetAction{
			Priority:     priority,
			Action:       "Eliminate negative cash flow by reducing expenses or increasing income",
			Category:     "critical",
			ImpactAmount: math.Abs(snapshot.NetMonthlyCashFlow),
			Timeframe:    "Immediately",
			Difficulty:   "hard",
		})
		priority++
	}

	// Priority 2: Build emergency fund
	if snapshot.RunwayMonths < 3 {
		targetEmergency := snapshot.MonthlyExpenses * 3
		needed := targetEmergency - snapshot.SavingsBalance
		actions = append(actions, BudgetAction{
			Priority:     priority,
			Action:       fmt.Sprintf("Build emergency fund to $%.2f (3 months expenses)", targetEmergency),
			Category:     "savings",
			ImpactAmount: needed,
			Timeframe:    "3-6 months",
			Difficulty:   "medium",
		})
		priority++
	}

	// Priority 3: Top optimization opportunities
	for i, opt := range optimizations {
		if i >= 3 {
			break
		}
		actions = append(actions, BudgetAction{
			Priority:     priority,
			Action:       opt.Suggestion,
			Category:     opt.Category,
			ImpactAmount: opt.PotentialSavings,
			Timeframe:    "This month",
			Difficulty:   opt.Difficulty,
		})
		priority++
	}

	// Priority 4: Goal-specific actions
	switch goal {
	case "build_wealth":
		actions = append(actions, BudgetAction{
			Priority:     priority,
			Action:       "Set up automatic monthly investment transfers",
			Category:     "investments",
			ImpactAmount: snapshot.MonthlyIncome * 0.15,
			Timeframe:    "This week",
			Difficulty:   "easy",
		})
	case "retire_early":
		actions = append(actions, BudgetAction{
			Priority:     priority,
			Action:       "Maximize tax-advantaged accounts and increase savings rate to 30%+",
			Category:     "retirement",
			ImpactAmount: snapshot.MonthlyIncome * 0.30,
			Timeframe:    "This month",
			Difficulty:   "medium",
		})
	case "save_for_emergency":
		actions = append(actions, BudgetAction{
			Priority:     priority,
			Action:       "Open a high-yield savings account and automate deposits",
			Category:     "savings",
			ImpactAmount: snapshot.MonthlyIncome * 0.20,
			Timeframe:    "This week",
			Difficulty:   "easy",
		})
	}

	return actions
}

func calculateBudgetScore(snapshot FinancialSnapshot, categorySpending []CategorySpending, goal string) (int, string) {
	score := 0

	// Savings rate (0-30 points)
	if snapshot.SavingsRate >= 20 {
		score += 30
	} else if snapshot.SavingsRate >= 10 {
		score += 20
	} else if snapshot.SavingsRate > 0 {
		score += 10
	}

	// Emergency fund (0-25 points)
	if snapshot.RunwayMonths >= 6 {
		score += 25
	} else if snapshot.RunwayMonths >= 3 {
		score += 15
	} else if snapshot.RunwayMonths >= 1 {
		score += 5
	}

	// Cash flow (0-25 points)
	if snapshot.NetMonthlyCashFlow > snapshot.MonthlyIncome*0.2 {
		score += 25
	} else if snapshot.NetMonthlyCashFlow > 0 {
		score += 15
	} else {
		score += 0 // Negative cash flow
	}

	// Spending discipline (0-20 points)
	essentialsPct := 0.0
	for _, cat := range categorySpending {
		if cat.Category == "bills" || cat.Category == "essentials" {
			essentialsPct += cat.Percentage
		}
	}
	if essentialsPct <= 50 {
		score += 20
	} else if essentialsPct <= 70 {
		score += 10
	}

	// Determine grade
	grade := "F"
	switch {
	case score >= 90:
		grade = "A"
	case score >= 80:
		grade = "B"
	case score >= 70:
		grade = "C"
	case score >= 60:
		grade = "D"
	}

	return score, grade
}

func generateSummaryInsight(snapshot FinancialSnapshot, forecast WealthForecast, score int, goal string) string {
	var insight string

	if score >= 80 {
		insight = fmt.Sprintf("Excellent financial health! You're saving %.1f%% of your income. ", snapshot.SavingsRate)
	} else if score >= 60 {
		insight = fmt.Sprintf("Good progress! Your %.1f%% savings rate is solid but could improve. ", snapshot.SavingsRate)
	} else {
		insight = fmt.Sprintf("Your finances need attention. Currently saving %.1f%% of income. ", snapshot.SavingsRate)
	}

	if len(forecast.ProjectedNetWorth) > 0 {
		lastProjection := forecast.ProjectedNetWorth[len(forecast.ProjectedNetWorth)-1]
		insight += fmt.Sprintf("At your current pace, you'll have $%.0f in %s. ", lastProjection.Amount, lastProjection.Period)
	}

	switch goal {
	case "build_wealth":
		insight += "Focus on maximizing your savings rate and investing consistently."
	case "retire_early":
		insight += "Consider aggressive savings (30%+) and low-cost index fund investments."
	case "save_for_emergency":
		insight += fmt.Sprintf("You have %.1f months of runway. Target is 3-6 months.", snapshot.RunwayMonths)
	default:
		insight += "Small consistent improvements will compound significantly over time."
	}

	return insight
}

// ============================================================================
// SPENDING TRACKERS - Daily, Weekly, Monthly
// ============================================================================

// buildSpendingTrackers creates comprehensive spending trackers
func buildSpendingTrackers(transactions []Transaction, monthlyBudget float64) SpendingTrackers {
	now := time.Now()

	// Calculate daily and weekly budgets from monthly
	dailyBudget := monthlyBudget / 30
	weeklyBudget := monthlyBudget / 4

	return SpendingTrackers{
		Daily:   buildDailyTracker(transactions, now, dailyBudget),
		Weekly:  buildWeeklyTracker(transactions, now, weeklyBudget),
		Monthly: buildMonthlyTracker(transactions, now, monthlyBudget),
	}
}

// buildDailyTracker creates the daily spending tracker
func buildDailyTracker(transactions []Transaction, now time.Time, dailyBudget float64) DailyTracker {
	// Get today and yesterday
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	// Build last 7 days
	last7Days := make([]DaySpending, 7)
	var totalSpent float64
	highestSpend := 0.0
	highestSpendDay := ""

	for i := 0; i < 7; i++ {
		day := today.AddDate(0, 0, -i)
		daySpending := buildDaySpending(transactions, day)
		last7Days[i] = daySpending
		totalSpent += daySpending.TotalSpent

		if daySpending.TotalSpent > highestSpend {
			highestSpend = daySpending.TotalSpent
			highestSpendDay = daySpending.DayOfWeek
		}
	}

	dailyAverage := totalSpent / 7
	todaySpending := buildDaySpending(transactions, today)

	// Determine today vs budget status
	todayVsBudget := "on_track"
	if todaySpending.TotalSpent > dailyBudget*1.2 {
		todayVsBudget = "over"
	} else if todaySpending.TotalSpent < dailyBudget*0.8 {
		todayVsBudget = "under"
	}

	return DailyTracker{
		Today:           todaySpending,
		Yesterday:       buildDaySpending(transactions, yesterday),
		Last7Days:       last7Days,
		DailyAverage:    dailyAverage,
		DailyBudget:     dailyBudget,
		TodayVsBudget:   todayVsBudget,
		HighestSpendDay: highestSpendDay,
	}
}

// buildDaySpending creates spending data for a single day
func buildDaySpending(transactions []Transaction, day time.Time) DaySpending {
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	var totalSpent, totalReceived float64
	var txCount int
	categoryAmounts := make(map[string]float64)
	var dayTransactions []TransactionDetail

	for _, tx := range transactions {
		if tx.CreatedAt.After(dayStart) && tx.CreatedAt.Before(dayEnd) {
			txCount++
			category := categorizeTransaction(tx)

			detail := TransactionDetail{
				ID:        tx.ID,
				Date:      tx.CreatedAt.Format("2006-01-02"),
				Time:      tx.CreatedAt.Format("15:04"),
				Type:      tx.Type,
				Amount:    tx.Amount,
				Category:  category,
				Recipient: tx.Recipient,
				Sender:    tx.Sender,
				Note:      tx.Note,
				DayOfWeek: tx.CreatedAt.Weekday().String(),
			}

			if tx.Type == "send" {
				totalSpent += tx.Amount
				categoryAmounts[category] += tx.Amount
			} else if tx.Type == "receive" {
				totalReceived += tx.Amount
			}

			dayTransactions = append(dayTransactions, detail)
		}
	}

	// Find top category
	topCategory := ""
	topAmount := 0.0
	for cat, amount := range categoryAmounts {
		if amount > topAmount {
			topAmount = amount
			topCategory = cat
		}
	}

	return DaySpending{
		Date:             day.Format("2006-01-02"),
		DayOfWeek:        day.Weekday().String(),
		TotalSpent:       totalSpent,
		TotalReceived:    totalReceived,
		NetFlow:          totalReceived - totalSpent,
		TransactionCount: txCount,
		TopCategory:      topCategory,
		Transactions:     dayTransactions,
	}
}

// buildWeeklyTracker creates the weekly spending tracker
func buildWeeklyTracker(transactions []Transaction, now time.Time, weeklyBudget float64) WeeklyTracker {
	// Find start of current week (Sunday)
	weekday := int(now.Weekday())
	currentWeekStart := now.AddDate(0, 0, -weekday)
	currentWeekStart = time.Date(currentWeekStart.Year(), currentWeekStart.Month(), currentWeekStart.Day(), 0, 0, 0, 0, now.Location())

	// Build last 4 weeks
	last4Weeks := make([]WeekSpending, 4)
	var totalSpent float64
	bestWeek := ""
	worstWeek := ""
	bestAmount := math.MaxFloat64
	worstAmount := 0.0

	for i := 0; i < 4; i++ {
		weekStart := currentWeekStart.AddDate(0, 0, -7*i)
		weekSpending := buildWeekSpending(transactions, weekStart, i)
		last4Weeks[i] = weekSpending
		totalSpent += weekSpending.TotalSpent

		if weekSpending.TotalSpent < bestAmount && weekSpending.TotalSpent > 0 {
			bestAmount = weekSpending.TotalSpent
			bestWeek = weekSpending.WeekStart
		}
		if weekSpending.TotalSpent > worstAmount {
			worstAmount = weekSpending.TotalSpent
			worstWeek = weekSpending.WeekStart
		}
	}

	weeklyAverage := totalSpent / 4
	currentWeek := last4Weeks[0]
	lastWeek := last4Weeks[1]

	// Calculate week over week change
	weekOverWeekChange := 0.0
	if lastWeek.TotalSpent > 0 {
		weekOverWeekChange = ((currentWeek.TotalSpent - lastWeek.TotalSpent) / lastWeek.TotalSpent) * 100
	}

	// Determine current vs budget status
	currentVsBudget := "on_track"
	if currentWeek.TotalSpent > weeklyBudget*1.2 {
		currentVsBudget = "over"
	} else if currentWeek.TotalSpent < weeklyBudget*0.8 {
		currentVsBudget = "under"
	}

	return WeeklyTracker{
		CurrentWeek:        currentWeek,
		LastWeek:           lastWeek,
		Last4Weeks:         last4Weeks,
		WeeklyAverage:      weeklyAverage,
		WeeklyBudget:       weeklyBudget,
		CurrentVsBudget:    currentVsBudget,
		WeekOverWeekChange: weekOverWeekChange,
		BestWeek:           bestWeek,
		WorstWeek:          worstWeek,
	}
}

// buildWeekSpending creates spending data for a week
func buildWeekSpending(transactions []Transaction, weekStart time.Time, weekNum int) WeekSpending {
	weekEnd := weekStart.AddDate(0, 0, 7)

	var totalSpent, totalReceived float64
	var txCount int
	categoryBreakdown := make(map[string]float64)

	for _, tx := range transactions {
		if tx.CreatedAt.After(weekStart) && tx.CreatedAt.Before(weekEnd) {
			txCount++
			category := categorizeTransaction(tx)

			if tx.Type == "send" {
				totalSpent += tx.Amount
				categoryBreakdown[category] += tx.Amount
			} else if tx.Type == "receive" {
				totalReceived += tx.Amount
			}
		}
	}

	// Find top category
	topCategory := ""
	topAmount := 0.0
	for cat, amount := range categoryBreakdown {
		if amount > topAmount {
			topAmount = amount
			topCategory = cat
		}
	}

	_, weekNumber := weekStart.ISOWeek()

	return WeekSpending{
		WeekStart:         weekStart.Format("2006-01-02"),
		WeekEnd:           weekEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		WeekNumber:        weekNumber,
		TotalSpent:        totalSpent,
		TotalReceived:     totalReceived,
		NetFlow:           totalReceived - totalSpent,
		TransactionCount:  txCount,
		CategoryBreakdown: categoryBreakdown,
		TopCategory:       topCategory,
	}
}

// buildMonthlyTracker creates the monthly spending tracker
func buildMonthlyTracker(transactions []Transaction, now time.Time, monthlyBudget float64) MonthlyTracker {
	// Get start of current month
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Build last 6 months
	last6Months := make([]MonthSpending, 6)
	var totalSpent float64

	for i := 0; i < 6; i++ {
		monthStart := currentMonthStart.AddDate(0, -i, 0)
		monthSpending := buildMonthSpending(transactions, monthStart)
		last6Months[i] = monthSpending
		totalSpent += monthSpending.TotalSpent
	}

	monthlyAverage := totalSpent / 6
	currentMonth := last6Months[0]
	lastMonth := last6Months[1]

	// Calculate month over month change
	monthOverMonthChange := 0.0
	if lastMonth.TotalSpent > 0 {
		monthOverMonthChange = ((currentMonth.TotalSpent - lastMonth.TotalSpent) / lastMonth.TotalSpent) * 100
	}

	// Calculate days remaining and projected spending
	daysInMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	dayOfMonth := now.Day()
	daysRemaining := daysInMonth - dayOfMonth

	dailySpendRate := currentMonth.TotalSpent / float64(dayOfMonth)
	projectedEndOfMonth := currentMonth.TotalSpent + (dailySpendRate * float64(daysRemaining))

	// Calculate remaining daily budget
	remainingBudget := monthlyBudget - currentMonth.TotalSpent
	dailyBudgetRemaining := 0.0
	if daysRemaining > 0 {
		dailyBudgetRemaining = remainingBudget / float64(daysRemaining)
	}

	// Determine current vs budget status
	expectedSpentSoFar := (monthlyBudget / float64(daysInMonth)) * float64(dayOfMonth)
	currentVsBudget := "on_track"
	if currentMonth.TotalSpent > expectedSpentSoFar*1.2 {
		currentVsBudget = "over"
	} else if currentMonth.TotalSpent < expectedSpentSoFar*0.8 {
		currentVsBudget = "under"
	}

	return MonthlyTracker{
		CurrentMonth:         currentMonth,
		LastMonth:            lastMonth,
		Last6Months:          last6Months,
		MonthlyAverage:       monthlyAverage,
		MonthlyBudget:        monthlyBudget,
		CurrentVsBudget:      currentVsBudget,
		MonthOverMonthChange: monthOverMonthChange,
		ProjectedEndOfMonth:  projectedEndOfMonth,
		DaysRemaining:        daysRemaining,
		DailyBudgetRemaining: dailyBudgetRemaining,
	}
}

// buildMonthSpending creates spending data for a month
func buildMonthSpending(transactions []Transaction, monthStart time.Time) MonthSpending {
	monthEnd := monthStart.AddDate(0, 1, 0)

	var totalSpent, totalReceived float64
	var txCount int
	categoryBreakdown := make(map[string]float64)

	for _, tx := range transactions {
		if tx.CreatedAt.After(monthStart) && tx.CreatedAt.Before(monthEnd) {
			txCount++
			category := categorizeTransaction(tx)

			if tx.Type == "send" {
				totalSpent += tx.Amount
				categoryBreakdown[category] += tx.Amount
			} else if tx.Type == "receive" {
				totalReceived += tx.Amount
			}
		}
	}

	// Calculate savings rate
	savingsRate := 0.0
	if totalReceived > 0 {
		savingsRate = ((totalReceived - totalSpent) / totalReceived) * 100
	}

	// Build top categories
	var topCategories []CategoryAmount
	for cat, amount := range categoryBreakdown {
		pct := 0.0
		if totalSpent > 0 {
			pct = (amount / totalSpent) * 100
		}
		topCategories = append(topCategories, CategoryAmount{
			Category: cat,
			Amount:   amount,
			Percent:  pct,
		})
	}
	// Sort by amount descending
	sort.Slice(topCategories, func(i, j int) bool {
		return topCategories[i].Amount > topCategories[j].Amount
	})
	if len(topCategories) > 5 {
		topCategories = topCategories[:5]
	}

	return MonthSpending{
		Month:             monthStart.Format("January 2006"),
		MonthShort:        monthStart.Format("Jan"),
		Year:              monthStart.Year(),
		TotalSpent:        totalSpent,
		TotalReceived:     totalReceived,
		NetFlow:           totalReceived - totalSpent,
		SavingsRate:       savingsRate,
		TransactionCount:  txCount,
		CategoryBreakdown: categoryBreakdown,
		TopCategories:     topCategories,
	}
}

// buildRecentTransactions creates a list of recent transactions with context
func buildRecentTransactions(transactions []Transaction, monthlyAvg float64) []TransactionDetail {
	// Sort transactions by date descending
	sortedTx := make([]Transaction, len(transactions))
	copy(sortedTx, transactions)
	sort.Slice(sortedTx, func(i, j int) bool {
		return sortedTx[i].CreatedAt.After(sortedTx[j].CreatedAt)
	})

	// Calculate average transaction for "large" detection
	avgTx := 0.0
	if len(sortedTx) > 0 {
		total := 0.0
		for _, tx := range sortedTx {
			if tx.Type == "send" {
				total += tx.Amount
			}
		}
		avgTx = total / float64(len(sortedTx))
	}

	// Build transaction details (last 20)
	var recentTx []TransactionDetail
	recipientCounts := make(map[string]int)

	// First pass: count recipients to detect recurring
	for _, tx := range sortedTx {
		if tx.Recipient != "" {
			recipientCounts[tx.Recipient]++
		}
	}

	limit := 20
	if len(sortedTx) < limit {
		limit = len(sortedTx)
	}

	for i := 0; i < limit; i++ {
		tx := sortedTx[i]
		category := categorizeTransaction(tx)

		isLarge := tx.Amount > avgTx*2 && tx.Type == "send"
		isRecurring := recipientCounts[tx.Recipient] >= 2

		recentTx = append(recentTx, TransactionDetail{
			ID:          tx.ID,
			Date:        tx.CreatedAt.Format("2006-01-02"),
			Time:        tx.CreatedAt.Format("15:04"),
			Type:        tx.Type,
			Amount:      tx.Amount,
			Category:    category,
			Recipient:   tx.Recipient,
			Sender:      tx.Sender,
			Note:        tx.Note,
			DayOfWeek:   tx.CreatedAt.Weekday().String(),
			IsLarge:     isLarge,
			IsRecurring: isRecurring,
		})
	}

	return recentTx
}

// GenerateBudgetPlanForAPI generates a budget plan for the HTTP API endpoint
// This is a convenience function that can be called directly without going through the tool system
func GenerateBudgetPlanForAPI(ctx context.Context, executor core.ToolExecutor, goal, riskTolerance string) BudgetPlan {
	// Set defaults
	if goal == "" {
		goal = "general_savings"
	}
	if riskTolerance == "" {
		riskTolerance = "moderate"
	}
	forecastYears := 5

	// Fetch financial data
	var balanceData, savingsData, ratesData map[string]interface{}
	var transactions []Transaction

	if DemoMode {
		balanceData = GenerateMockBalance()
		savingsData = GenerateMockSavings()
		ratesData = GenerateMockVaultRates()
		transactions = GenerateMockTransactions(90) // 3 months of data
	} else {
		// Fetch from Liminal API
		balanceResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_balance", Input: []byte("{}"), RequestID: "budget-api",
		})
		savingsResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_savings_balance", Input: []byte("{}"), RequestID: "budget-api",
		})
		ratesResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_vault_rates", Input: []byte("{}"), RequestID: "budget-api",
		})
		txReq, _ := json.Marshal(map[string]interface{}{"limit": 500})
		txResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_transactions", Input: txReq, RequestID: "budget-api",
		})

		if balanceResp != nil && balanceResp.Success {
			json.Unmarshal(balanceResp.Data, &balanceData)
		}
		if savingsResp != nil && savingsResp.Success {
			json.Unmarshal(savingsResp.Data, &savingsData)
		}
		if ratesResp != nil && ratesResp.Success {
			json.Unmarshal(ratesResp.Data, &ratesData)
		}
		if txResp != nil && txResp.Success {
			var txData struct{ Transactions []Transaction }
			json.Unmarshal(txResp.Data, &txData)
			transactions = txData.Transactions
		}
	}

	// Generate the budget plan
	return generateBudgetPlan(
		balanceData, savingsData, ratesData, transactions,
		goal, 0, riskTolerance, forecastYears,
	)
}
