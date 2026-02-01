// Smart Savings - AI-driven savings optimization
// Provides tools for Claude to analyze and optimize user's savings
package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// ============================================================================
// SMART SAVINGS DATA STRUCTURES
// ============================================================================

// VaultPosition represents a user's position in a savings vault
type VaultPosition struct {
	VaultID      string  `json:"vault_id"`
	VaultName    string  `json:"vault_name"`
	Amount       float64 `json:"amount"`
	APY          float64 `json:"apy"`
	Risk         string  `json:"risk"`
	MonthlyYield float64 `json:"monthly_yield"`
	YearlyYield  float64 `json:"yearly_yield"`
	DepositedAt  string  `json:"deposited_at"`
}

// VaultOption represents an available vault for deposits
type VaultOption struct {
	VaultID       string  `json:"vault_id"`
	Name          string  `json:"name"`
	APY           float64 `json:"apy"`
	Risk          string  `json:"risk"`
	MinDeposit    float64 `json:"min_deposit"`
	Description   string  `json:"description"`
	Recommended   bool    `json:"recommended"`
	APYDifference float64 `json:"apy_difference,omitempty"` // vs current
}

// SavingsRecommendation is an actionable suggestion
type SavingsRecommendation struct {
	Type        string  `json:"type"`     // "sweep", "rebalance", "goal", "alert"
	Priority    string  `json:"priority"` // "high", "medium", "low"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Action      string  `json:"action,omitempty"`   // Suggested action text
	Amount      float64 `json:"amount,omitempty"`   // Relevant amount
	Impact      float64 `json:"impact,omitempty"`   // Projected yearly gain
	VaultID     string  `json:"vault_id,omitempty"` // Target vault if applicable
}

// SavingsGoal tracks progress toward a savings target
type SavingsGoal struct {
	Name            string  `json:"name"`
	TargetAmount    float64 `json:"target_amount"`
	CurrentAmount   float64 `json:"current_amount"`
	Progress        float64 `json:"progress"`         // 0-100
	Status          string  `json:"status"`           // "building", "funded", "overfunded"
	MonthlyRequired float64 `json:"monthly_required"` // To reach goal
	TimeToGoal      string  `json:"time_to_goal"`     // E.g., "4 months"
}

// SavingsAnalysis is the comprehensive analysis result
type SavingsAnalysis struct {
	// Current State
	WalletBalance    float64 `json:"wallet_balance"`
	TotalSavings     float64 `json:"total_savings"`
	TotalAssets      float64 `json:"total_assets"`
	IdleCash         float64 `json:"idle_cash"`          // Cash that could be earning
	MinWalletReserve float64 `json:"min_wallet_reserve"` // Recommended minimum

	// Current Yield
	CurrentWeightedAPY float64 `json:"current_weighted_apy"`
	MonthlyEarnings    float64 `json:"monthly_earnings"`
	YearlyEarnings     float64 `json:"yearly_earnings"`

	// Positions
	CurrentPositions []VaultPosition `json:"current_positions"`

	// Available Vaults
	AvailableVaults  []VaultOption `json:"available_vaults"`
	BestVaultForUser *VaultOption  `json:"best_vault_for_user,omitempty"`

	// Optimization Opportunity
	HasOpportunity       bool    `json:"has_opportunity"`
	SweepAmount          float64 `json:"sweep_amount"`
	RecommendedVault     string  `json:"recommended_vault,omitempty"`
	RecommendedVaultName string  `json:"recommended_vault_name,omitempty"`
	ProjectedNewAPY      float64 `json:"projected_new_apy"`
	ProjectedGainYearly  float64 `json:"projected_gain_yearly"`
	APYImprovement       float64 `json:"apy_improvement"`

	// Goals
	EmergencyFund SavingsGoal `json:"emergency_fund"`

	// Health Score
	SavingsHealthScore int    `json:"savings_health_score"` // 0-100
	SavingsHealthGrade string `json:"savings_health_grade"` // A-F

	// Recommendations
	Recommendations []SavingsRecommendation `json:"recommendations"`
	SummaryInsight  string                  `json:"summary_insight"`

	// Metadata
	AnalyzedAt    string `json:"analyzed_at"`
	RiskTolerance string `json:"risk_tolerance"`
}

// SweepResult is the result of executing a savings sweep
type SweepResult struct {
	Success           bool    `json:"success"`
	AmountDeposited   float64 `json:"amount_deposited"`
	VaultID           string  `json:"vault_id"`
	VaultName         string  `json:"vault_name"`
	NewAPY            float64 `json:"new_apy"`
	ProjectedEarnings float64 `json:"projected_yearly_earnings"`
	NewWalletBalance  float64 `json:"new_wallet_balance"`
	NewTotalSavings   float64 `json:"new_total_savings"`
	Message           string  `json:"message"`
}

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	DefaultMinWalletReserve = 200.0 // Keep at least $200 in wallet
	DefaultSweepPercentage  = 0.80  // Sweep 80% of idle cash by default
	EmergencyFundMonths     = 3.0   // Target 3 months of expenses
)

// Risk level descriptions
var riskDescriptions = map[string]string{
	"low":    "Conservative - FDIC insured or equivalent, minimal risk",
	"medium": "Moderate - Some market exposure, balanced risk/reward",
	"high":   "Aggressive - Higher yields with more volatility",
}

// ============================================================================
// ANALYSIS FUNCTIONS
// ============================================================================

// analyzeSavings performs comprehensive savings analysis
func analyzeSavings(
	balance map[string]interface{},
	savings map[string]interface{},
	rates map[string]interface{},
	transactions []Transaction,
	riskTolerance string,
) SavingsAnalysis {
	// Extract current balances
	walletBalance := extractBalance(balance)
	savingsBalance, positions := extractSavingsPositions(savings)
	vaultOptions := extractVaultOptions(rates, riskTolerance)

	// Calculate current weighted APY
	currentAPY := calculateWeightedAPY(positions)

	// Calculate monthly expenses from transactions
	monthlyExpenses := calculateMonthlyExpenses(transactions)

	// Calculate idle cash (wallet balance minus reserve)
	minReserve := DefaultMinWalletReserve
	idleCash := math.Max(0, walletBalance-minReserve)

	// Find best vault for user's risk tolerance
	var bestVault *VaultOption
	for i := range vaultOptions {
		if vaultOptions[i].Recommended {
			bestVault = &vaultOptions[i]
			break
		}
	}

	// Calculate sweep opportunity
	sweepAmount := idleCash * DefaultSweepPercentage
	hasOpportunity := sweepAmount >= 10 && bestVault != nil

	var projectedGain, projectedNewAPY, apyImprovement float64
	var recommendedVault, recommendedVaultName string

	if hasOpportunity && bestVault != nil {
		recommendedVault = bestVault.VaultID
		recommendedVaultName = bestVault.Name
		projectedGain = sweepAmount * (bestVault.APY / 100)

		// Calculate new weighted APY after sweep
		newTotal := savingsBalance + sweepAmount
		if newTotal > 0 {
			projectedNewAPY = ((savingsBalance * currentAPY) + (sweepAmount * bestVault.APY)) / newTotal
		}
		apyImprovement = projectedNewAPY - currentAPY
	}

	// Calculate emergency fund progress
	emergencyTarget := monthlyExpenses * EmergencyFundMonths
	if emergencyTarget < 1000 {
		emergencyTarget = 5000 // Default minimum
	}
	emergencyProgress := 0.0
	emergencyStatus := "building"
	if emergencyTarget > 0 {
		emergencyProgress = (savingsBalance / emergencyTarget) * 100
		if emergencyProgress >= 100 {
			emergencyStatus = "funded"
			if emergencyProgress > 150 {
				emergencyStatus = "overfunded"
			}
		}
	}

	monthlyRequired := 0.0
	timeToGoal := "Funded!"
	if emergencyProgress < 100 {
		remaining := emergencyTarget - savingsBalance
		monthlyRequired = remaining / 6 // 6 months to fund
		months := int(remaining / math.Max(monthlyRequired, 100))
		timeToGoal = fmt.Sprintf("%d months", months)
	}

	// Calculate health score
	healthScore, healthGrade := calculateSavingsHealthScore(
		walletBalance, savingsBalance, idleCash, currentAPY,
		emergencyProgress, monthlyExpenses,
	)

	// Generate recommendations
	recommendations := generateSavingsRecommendations(
		idleCash, sweepAmount, currentAPY, bestVault,
		emergencyProgress, positions, vaultOptions,
	)

	// Generate summary insight
	summaryInsight := generateSavingsSummary(
		walletBalance, savingsBalance, idleCash, currentAPY,
		hasOpportunity, sweepAmount, projectedGain, bestVault,
	)

	return SavingsAnalysis{
		WalletBalance:        walletBalance,
		TotalSavings:         savingsBalance,
		TotalAssets:          walletBalance + savingsBalance,
		IdleCash:             idleCash,
		MinWalletReserve:     minReserve,
		CurrentWeightedAPY:   currentAPY,
		MonthlyEarnings:      savingsBalance * (currentAPY / 100) / 12,
		YearlyEarnings:       savingsBalance * (currentAPY / 100),
		CurrentPositions:     positions,
		AvailableVaults:      vaultOptions,
		BestVaultForUser:     bestVault,
		HasOpportunity:       hasOpportunity,
		SweepAmount:          sweepAmount,
		RecommendedVault:     recommendedVault,
		RecommendedVaultName: recommendedVaultName,
		ProjectedNewAPY:      projectedNewAPY,
		ProjectedGainYearly:  projectedGain,
		APYImprovement:       apyImprovement,
		EmergencyFund: SavingsGoal{
			Name:            "Emergency Fund",
			TargetAmount:    emergencyTarget,
			CurrentAmount:   savingsBalance,
			Progress:        math.Min(emergencyProgress, 100),
			Status:          emergencyStatus,
			MonthlyRequired: monthlyRequired,
			TimeToGoal:      timeToGoal,
		},
		SavingsHealthScore: healthScore,
		SavingsHealthGrade: healthGrade,
		Recommendations:    recommendations,
		SummaryInsight:     summaryInsight,
		AnalyzedAt:         time.Now().Format(time.RFC3339),
		RiskTolerance:      riskTolerance,
	}
}

// extractSavingsPositions extracts vault positions from savings data
func extractSavingsPositions(savings map[string]interface{}) (float64, []VaultPosition) {
	var positions []VaultPosition
	totalBalance := 0.0

	if savings == nil {
		return totalBalance, positions
	}

	// Try to get total balance
	if tb, ok := savings["total_balance"].(float64); ok {
		totalBalance = tb
	}

	// Extract positions
	if positionsData, ok := savings["positions"].([]interface{}); ok {
		for _, p := range positionsData {
			if pos, ok := p.(map[string]interface{}); ok {
				amount := 0.0
				if a, ok := pos["amount"].(float64); ok {
					amount = a
				}
				apy := 0.0
				if a, ok := pos["apy"].(float64); ok {
					apy = a
				}

				vaultID := ""
				if v, ok := pos["vault_id"].(string); ok {
					vaultID = v
				}
				vaultName := ""
				if v, ok := pos["vault_name"].(string); ok {
					vaultName = v
				}
				risk := "low"
				if r, ok := pos["risk"].(string); ok {
					risk = r
				}
				depositedAt := ""
				if d, ok := pos["deposited_at"].(string); ok {
					depositedAt = d
				}

				positions = append(positions, VaultPosition{
					VaultID:      vaultID,
					VaultName:    vaultName,
					Amount:       amount,
					APY:          apy,
					Risk:         risk,
					MonthlyYield: amount * (apy / 100) / 12,
					YearlyYield:  amount * (apy / 100),
					DepositedAt:  depositedAt,
				})

				if totalBalance == 0 {
					totalBalance += amount
				}
			}
		}
	}

	return totalBalance, positions
}

// extractVaultOptions extracts available vaults filtered by risk tolerance
func extractVaultOptions(rates map[string]interface{}, riskTolerance string) []VaultOption {
	var options []VaultOption

	if rates == nil {
		// Return demo vaults
		return getDemoVaults(riskTolerance)
	}

	// Try to extract from vault_rates array
	if vaultRates, ok := rates["vault_rates"].([]interface{}); ok {
		for _, v := range vaultRates {
			if vault, ok := v.(map[string]interface{}); ok {
				vaultID := ""
				if id, ok := vault["vault_id"].(string); ok {
					vaultID = id
				}
				name := ""
				if n, ok := vault["name"].(string); ok {
					name = n
				}
				apy := 0.0
				if a, ok := vault["apy"].(float64); ok {
					apy = a
				}
				risk := "low"
				if r, ok := vault["risk"].(string); ok {
					risk = r
				}
				minDeposit := 0.0
				if m, ok := vault["min_deposit"].(float64); ok {
					minDeposit = m
				}

				// Filter by risk tolerance
				if !matchesRiskTolerance(risk, riskTolerance) {
					continue
				}

				options = append(options, VaultOption{
					VaultID:     vaultID,
					Name:        name,
					APY:         apy,
					Risk:        risk,
					MinDeposit:  minDeposit,
					Description: riskDescriptions[risk],
				})
			}
		}
	}

	if len(options) == 0 {
		return getDemoVaults(riskTolerance)
	}

	// Sort by APY descending and mark best as recommended
	sort.Slice(options, func(i, j int) bool {
		return options[i].APY > options[j].APY
	})

	if len(options) > 0 {
		options[0].Recommended = true
	}

	return options
}

// getDemoVaults returns demo vault options
func getDemoVaults(riskTolerance string) []VaultOption {
	allVaults := []VaultOption{
		{
			VaultID:     "vault_001",
			Name:        "High Yield Savings",
			APY:         4.5,
			Risk:        "low",
			MinDeposit:  10,
			Description: riskDescriptions["low"],
		},
		{
			VaultID:     "vault_002",
			Name:        "Growth Fund",
			APY:         6.2,
			Risk:        "medium",
			MinDeposit:  100,
			Description: riskDescriptions["medium"],
		},
		{
			VaultID:     "vault_003",
			Name:        "Aggressive Growth",
			APY:         9.5,
			Risk:        "high",
			MinDeposit:  500,
			Description: riskDescriptions["high"],
		},
	}

	var filtered []VaultOption
	for _, v := range allVaults {
		if matchesRiskTolerance(v.Risk, riskTolerance) {
			filtered = append(filtered, v)
		}
	}

	// Sort by APY and mark best as recommended
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].APY > filtered[j].APY
	})

	if len(filtered) > 0 {
		filtered[0].Recommended = true
	}

	return filtered
}

// matchesRiskTolerance checks if a vault's risk matches user tolerance
func matchesRiskTolerance(vaultRisk, userTolerance string) bool {
	riskLevels := map[string]int{"low": 1, "medium": 2, "high": 3}
	vaultLevel := riskLevels[vaultRisk]
	userLevel := riskLevels[userTolerance]
	if userLevel == 0 {
		userLevel = 1 // Default to low risk
	}
	return vaultLevel <= userLevel
}

// calculateWeightedAPY calculates the weighted average APY across positions
func calculateWeightedAPY(positions []VaultPosition) float64 {
	if len(positions) == 0 {
		return 0
	}

	totalValue := 0.0
	weightedSum := 0.0

	for _, pos := range positions {
		totalValue += pos.Amount
		weightedSum += pos.Amount * pos.APY
	}

	if totalValue == 0 {
		return 0
	}

	return weightedSum / totalValue
}

// calculateMonthlyExpenses estimates monthly expenses from transactions
func calculateMonthlyExpenses(transactions []Transaction) float64 {
	if len(transactions) == 0 {
		return 1500 // Default estimate
	}

	// Sum expenses from last 30 days
	now := time.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	totalExpenses := 0.0
	for _, tx := range transactions {
		if tx.Type == "send" && tx.CreatedAt.After(thirtyDaysAgo) {
			totalExpenses += tx.Amount
		}
	}

	if totalExpenses < 500 {
		return 1500 // Minimum estimate
	}

	return totalExpenses
}

// calculateSavingsHealthScore computes the savings health score
func calculateSavingsHealthScore(
	walletBalance, savingsBalance, idleCash, currentAPY,
	emergencyProgress, monthlyExpenses float64,
) (int, string) {
	score := 50 // Start at 50

	// Emergency fund progress (+30 points max)
	if emergencyProgress >= 100 {
		score += 30
	} else {
		score += int(emergencyProgress * 0.3)
	}

	// Savings rate - compare savings to expenses (+20 points max)
	savingsRatio := savingsBalance / math.Max(monthlyExpenses*3, 1)
	if savingsRatio >= 1 {
		score += 20
	} else {
		score += int(savingsRatio * 20)
	}

	// APY optimization (+15 points max)
	if currentAPY >= 5 {
		score += 15
	} else if currentAPY >= 3 {
		score += 10
	} else if currentAPY >= 1 {
		score += 5
	}

	// Idle cash penalty (-15 points max)
	if idleCash > 500 {
		score -= 15
	} else if idleCash > 200 {
		score -= 10
	} else if idleCash > 100 {
		score -= 5
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
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

// generateSavingsRecommendations creates actionable recommendations
func generateSavingsRecommendations(
	idleCash, sweepAmount, currentAPY float64,
	bestVault *VaultOption,
	emergencyProgress float64,
	positions []VaultPosition,
	vaults []VaultOption,
) []SavingsRecommendation {
	var recommendations []SavingsRecommendation

	// Sweep opportunity
	if sweepAmount >= 10 && bestVault != nil {
		projectedGain := sweepAmount * (bestVault.APY / 100)
		recommendations = append(recommendations, SavingsRecommendation{
			Type:        "sweep",
			Priority:    "high",
			Title:       fmt.Sprintf("Move $%.0f to %s", sweepAmount, bestVault.Name),
			Description: fmt.Sprintf("You have $%.2f idle in your wallet. Move it to %s earning %.1f%% APY to earn ~$%.2f more per year.", idleCash, bestVault.Name, bestVault.APY, projectedGain),
			Action:      fmt.Sprintf("Deposit $%.0f", sweepAmount),
			Amount:      sweepAmount,
			Impact:      projectedGain,
			VaultID:     bestVault.VaultID,
		})
	}

	// Emergency fund recommendation
	if emergencyProgress < 100 {
		recommendations = append(recommendations, SavingsRecommendation{
			Type:        "goal",
			Priority:    "medium",
			Title:       "Build your emergency fund",
			Description: fmt.Sprintf("Your emergency fund is %.0f%% funded. Aim for 3-6 months of expenses.", emergencyProgress),
			Action:      "Increase monthly savings",
		})
	}

	// Better rate available
	if len(positions) > 0 && len(vaults) > 0 {
		currentBestAPY := 0.0
		for _, pos := range positions {
			if pos.APY > currentBestAPY {
				currentBestAPY = pos.APY
			}
		}
		if vaults[0].APY > currentBestAPY+0.5 {
			recommendations = append(recommendations, SavingsRecommendation{
				Type:        "rebalance",
				Priority:    "low",
				Title:       fmt.Sprintf("Higher rate available: %.1f%% APY", vaults[0].APY),
				Description: fmt.Sprintf("%s offers %.1f%% APY vs your current %.1f%%. Consider rebalancing.", vaults[0].Name, vaults[0].APY, currentBestAPY),
				VaultID:     vaults[0].VaultID,
			})
		}
	}

	// Low APY alert
	if currentAPY < 2 && len(vaults) > 0 {
		recommendations = append(recommendations, SavingsRecommendation{
			Type:        "alert",
			Priority:    "medium",
			Title:       "Your savings could earn more",
			Description: fmt.Sprintf("Your current APY is only %.1f%%. You could earn up to %.1f%% with %s.", currentAPY, vaults[0].APY, vaults[0].Name),
		})
	}

	return recommendations
}

// generateSavingsSummary creates a human-readable summary
func generateSavingsSummary(
	walletBalance, savingsBalance, idleCash, currentAPY float64,
	hasOpportunity bool, sweepAmount, projectedGain float64,
	bestVault *VaultOption,
) string {
	summary := fmt.Sprintf("You have $%.2f in savings earning %.1f%% APY ($%.2f/year).",
		savingsBalance, currentAPY, savingsBalance*(currentAPY/100))

	if hasOpportunity && bestVault != nil {
		summary += fmt.Sprintf(" You have $%.2f idle cash that could be earning %.1f%% in %s - that's ~$%.2f more per year!",
			idleCash, bestVault.APY, bestVault.Name, projectedGain)
	} else if idleCash < 50 {
		summary += " Great job keeping your idle cash minimized!"
	}

	return summary
}

// ============================================================================
// AI TOOLS
// ============================================================================

// CreateGetSavingsAnalysisTool creates the savings analysis tool
func CreateGetSavingsAnalysisTool(executor core.ToolExecutor) core.Tool {
	return tools.New("get_savings_analysis").
		Description("Analyze user's savings to find optimization opportunities. Returns current positions, available vaults, idle cash, APY comparison, emergency fund progress, and recommendations. Use this when user asks about savings, yields, or wants to optimize their money.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"risk_tolerance": tools.StringEnumProperty(
				"User's risk tolerance for vault recommendations",
				"low", "medium", "high",
			),
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

			// Fetch financial data
			var balanceData, savingsData, ratesData map[string]interface{}
			var transactions []Transaction

			if DemoMode {
				balanceData = GenerateMockBalance()
				savingsData = GenerateMockSavings()
				ratesData = GenerateMockVaultRates()
				transactions = GenerateMockTransactions(30)
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
				txReq, _ := json.Marshal(map[string]interface{}{"limit": 100})
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

			// Perform analysis
			analysis := analyzeSavings(balanceData, savingsData, ratesData, transactions, params.RiskTolerance)

			return &core.ToolResult{
				Success: true,
				Data:    analysis,
			}, nil
		}).
		Build()
}

// CreateExecuteSavingsSweepTool creates the savings sweep tool (requires confirmation)
func CreateExecuteSavingsSweepTool(executor core.ToolExecutor) core.Tool {
	return tools.New("execute_savings_sweep").
		Description("Move idle cash from wallet to a high-yield savings vault. REQUIRES USER CONFIRMATION. Use after analyzing savings with get_savings_analysis when user wants to optimize their money.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"amount":   tools.NumberProperty("Amount to deposit into savings (in USD)"),
			"vault_id": tools.StringProperty("Target vault ID (optional - uses best available if not specified)"),
		}, "amount")).
		RequiresConfirmation().
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				Amount  float64 `json:"amount"`
				VaultID string  `json:"vault_id"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				return &core.ToolResult{
					Success: false,
					Error:   "Invalid input parameters",
				}, nil
			}

			if params.Amount <= 0 {
				return &core.ToolResult{
					Success: false,
					Error:   "Amount must be greater than 0",
				}, nil
			}

			// Get current balances to validate
			var balanceData, savingsData, ratesData map[string]interface{}

			if DemoMode {
				balanceData = GenerateMockBalance()
				savingsData = GenerateMockSavings()
				ratesData = GenerateMockVaultRates()
			} else {
				balanceResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_balance", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				savingsResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_savings_balance", Input: []byte("{}"), RequestID: toolParams.RequestID,
				})
				ratesResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
					UserID: toolParams.UserID, Tool: "get_vault_rates", Input: []byte("{}"), RequestID: toolParams.RequestID,
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
			}

			walletBalance := extractBalance(balanceData)

			// Check if user has enough balance
			if params.Amount > walletBalance-DefaultMinWalletReserve {
				return &core.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("Insufficient funds. You have $%.2f available (keeping $%.2f reserve). Requested: $%.2f", walletBalance-DefaultMinWalletReserve, DefaultMinWalletReserve, params.Amount),
				}, nil
			}

			// Find target vault
			vaults := extractVaultOptions(ratesData, "high") // Get all vaults
			var targetVault *VaultOption

			if params.VaultID != "" {
				for i := range vaults {
					if vaults[i].VaultID == params.VaultID {
						targetVault = &vaults[i]
						break
					}
				}
			}

			if targetVault == nil && len(vaults) > 0 {
				targetVault = &vaults[0] // Use best available
			}

			if targetVault == nil {
				return &core.ToolResult{
					Success: false,
					Error:   "No suitable vault found for deposit",
				}, nil
			}

			// Check minimum deposit
			if params.Amount < targetVault.MinDeposit {
				return &core.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("Minimum deposit for %s is $%.2f", targetVault.Name, targetVault.MinDeposit),
				}, nil
			}

			// Execute the deposit
			if !DemoMode {
				// Call the actual Liminal deposit API
				depositReq, _ := json.Marshal(map[string]interface{}{
					"amount":   params.Amount,
					"vault_id": targetVault.VaultID,
				})
				depositResp, err := executor.Execute(ctx, &core.ExecuteRequest{
					UserID:    toolParams.UserID,
					Tool:      "deposit_savings",
					Input:     depositReq,
					RequestID: toolParams.RequestID,
				})
				if err != nil || (depositResp != nil && !depositResp.Success) {
					errMsg := "Deposit failed"
					if depositResp != nil && depositResp.Error != "" {
						errMsg = depositResp.Error
					}
					return &core.ToolResult{
						Success: false,
						Error:   errMsg,
					}, nil
				}
			}

			// Calculate new balances (simulated for demo)
			newWalletBalance := walletBalance - params.Amount
			savingsBalance, _ := extractSavingsPositions(savingsData)
			newTotalSavings := savingsBalance + params.Amount
			projectedYearlyEarnings := newTotalSavings * (targetVault.APY / 100)

			return &core.ToolResult{
				Success: true,
				Data: SweepResult{
					Success:           true,
					AmountDeposited:   params.Amount,
					VaultID:           targetVault.VaultID,
					VaultName:         targetVault.Name,
					NewAPY:            targetVault.APY,
					ProjectedEarnings: projectedYearlyEarnings,
					NewWalletBalance:  newWalletBalance,
					NewTotalSavings:   newTotalSavings,
					Message:           fmt.Sprintf("Successfully deposited $%.2f into %s earning %.1f%% APY. You'll earn approximately $%.2f per year!", params.Amount, targetVault.Name, targetVault.APY, projectedYearlyEarnings),
				},
			}, nil
		}).
		Build()
}

// GenerateSavingsAnalysisForAPI generates savings analysis for the HTTP API
func GenerateSavingsAnalysisForAPI(ctx context.Context, executor core.ToolExecutor, riskTolerance string) SavingsAnalysis {
	if riskTolerance == "" {
		riskTolerance = "low"
	}

	var balanceData, savingsData, ratesData map[string]interface{}
	var transactions []Transaction

	if DemoMode {
		balanceData = GenerateMockBalance()
		savingsData = GenerateMockSavings()
		ratesData = GenerateMockVaultRates()
		transactions = GenerateMockTransactions(30)
	} else {
		balanceResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_balance", Input: []byte("{}"), RequestID: "savings-api",
		})
		savingsResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_savings_balance", Input: []byte("{}"), RequestID: "savings-api",
		})
		ratesResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_vault_rates", Input: []byte("{}"), RequestID: "savings-api",
		})
		txReq, _ := json.Marshal(map[string]interface{}{"limit": 100})
		txResp, _ := executor.Execute(ctx, &core.ExecuteRequest{
			UserID: "user", Tool: "get_transactions", Input: txReq, RequestID: "savings-api",
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

	return analyzeSavings(balanceData, savingsData, ratesData, transactions, riskTolerance)
}
