// Package trading - Journey tools for the Stabilize → Save → Invest flow
package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
	"github.com/google/uuid"
)

// ============================================================================
// STEP 1: CHAT WITH NIM - Onboarding Tools
// ============================================================================

// CreateUserPlanTool creates/updates the user's financial plan during onboarding.
func CreateUserPlanTool(db *Database) core.Tool {
	return tools.New("create_user_plan").
		Description(`Create or update the user's financial plan during onboarding. Use this when the user shares their financial goals, income, and preferences.

This tool captures:
- Monthly savings target and emergency fund goal
- Risk profile (conservative, moderate, aggressive)  
- Income cadence (weekly, biweekly, monthly)
- Spending caps and automation preferences

Call this tool after gathering the user's goals through conversation.`).
		Schema(tools.ObjectSchema(map[string]interface{}{
			"monthly_savings_target": tools.NumberProperty("Target monthly savings amount"),
			"emergency_fund_target":  tools.NumberProperty("Target emergency fund amount (default: 3x monthly expenses)"),
			"investment_goal":        tools.StringEnumProperty("Primary investment goal", "grow_wealth", "retirement", "big_purchase", "passive_income"),
			"risk_profile":           tools.StringEnumProperty("Risk tolerance", "conservative", "moderate", "aggressive"),
			"income_cadence":         tools.StringEnumProperty("How often user gets paid", "weekly", "biweekly", "monthly", "irregular"),
			"income_date":            tools.IntegerProperty("Day of month when income arrives (1-31, for monthly)"),
			"estimated_income":       tools.NumberProperty("Estimated monthly income"),
			"buffer_amount":          tools.NumberProperty("Minimum buffer to keep in checking account"),
			"auto_sweep_enabled":     tools.BooleanProperty("Enable automatic surplus sweeps to savings"),
			"round_ups_enabled":      tools.BooleanProperty("Enable round-up savings on purchases"),
		}, "monthly_savings_target", "risk_profile", "income_cadence", "estimated_income")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				MonthlySavingsTarget float64 `json:"monthly_savings_target"`
				EmergencyFundTarget  float64 `json:"emergency_fund_target"`
				InvestmentGoal       string  `json:"investment_goal"`
				RiskProfile          string  `json:"risk_profile"`
				IncomeCadence        string  `json:"income_cadence"`
				IncomeDate           int     `json:"income_date"`
				EstimatedIncome      float64 `json:"estimated_income"`
				BufferAmount         float64 `json:"buffer_amount"`
				AutoSweepEnabled     bool    `json:"auto_sweep_enabled"`
				RoundUpsEnabled      bool    `json:"round_ups_enabled"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			userID := "user" // TODO: Get from context

			// Set defaults
			if params.InvestmentGoal == "" {
				params.InvestmentGoal = "grow_wealth"
			}
			if params.BufferAmount == 0 {
				params.BufferAmount = params.EstimatedIncome * 0.1 // 10% as default buffer
			}
			if params.EmergencyFundTarget == 0 {
				params.EmergencyFundTarget = params.EstimatedIncome * 3 // 3 months
			}

			plan := &UserPlan{
				UserID: userID,
				Goals: Goals{
					MonthlySavingsTarget: params.MonthlySavingsTarget,
					EmergencyFundTarget:  params.EmergencyFundTarget,
					InvestmentGoal:       params.InvestmentGoal,
				},
				Rules: FinancialRules{
					BufferAmount:        params.BufferAmount,
					InvestmentFrequency: "weekly",
					RiskProfile:         params.RiskProfile,
					AutoSweepEnabled:    params.AutoSweepEnabled,
					RoundUpsEnabled:     params.RoundUpsEnabled,
				},
				Income: IncomeProfile{
					Cadence:         params.IncomeCadence,
					ExpectedDate:    params.IncomeDate,
					EstimatedAmount: params.EstimatedIncome,
				},
				OnboardingComplete: true,
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}

			if err := db.SaveUserPlan(plan); err != nil {
				return nil, fmt.Errorf("failed to save plan: %w", err)
			}

			// Initialize emergency fund targets
			stage1, stage2, stage3 := CalculateEmergencyFundTargets(params.EstimatedIncome)
			ef := &EmergencyFund{
				UserID:          userID,
				Current:         0,
				Stage1Target:    stage1,
				Stage2Target:    stage2,
				Stage3Target:    stage3,
				MonthlyExpenses: params.EstimatedIncome * 0.8, // Assume 80% spent
			}
			db.SaveEmergencyFund(ef)

			return map[string]interface{}{
				"success": true,
				"message": "Financial plan created successfully!",
				"plan": map[string]interface{}{
					"monthly_savings_target": fmt.Sprintf("£%.2f", params.MonthlySavingsTarget),
					"emergency_fund_target":  fmt.Sprintf("£%.2f", params.EmergencyFundTarget),
					"risk_profile":           params.RiskProfile,
					"income_cadence":         params.IncomeCadence,
					"buffer_amount":          fmt.Sprintf("£%.2f", params.BufferAmount),
					"auto_sweep":             params.AutoSweepEnabled,
					"round_ups":              params.RoundUpsEnabled,
				},
				"next_step": "Now let's create your budget based on your income and goals.",
			}, nil
		}).
		Build()
}

// GetUserPlanTool retrieves the user's financial plan.
func GetUserPlanTool(db *Database) core.Tool {
	return tools.New("get_user_plan").
		Description("Get the user's current financial plan, goals, and rules. Use this to understand their setup before making recommendations.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			plan, err := db.GetUserPlan(userID)
			if err != nil {
				return nil, err
			}

			if plan == nil {
				return map[string]interface{}{
					"success":             false,
					"onboarding_complete": false,
					"message":             "No financial plan found. Let's set one up! What are your financial goals?",
				}, nil
			}

			return map[string]interface{}{
				"success":             true,
				"onboarding_complete": plan.OnboardingComplete,
				"goals":               plan.Goals,
				"rules":               plan.Rules,
				"income":              plan.Income,
			}, nil
		}).
		Build()
}

// GetJourneyStatusTool shows progress through the journey.
func GetJourneyStatusTool(db *Database) core.Tool {
	return tools.New("get_journey_status").
		Description("Get the user's progress through the financial journey (Chat → Budget → Subscriptions → Savings → Investing → Autopilot).").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			status, err := db.GetJourneyStatus(userID)
			if err != nil {
				return nil, err
			}

			return status, nil
		}).
		Build()
}

// ============================================================================
// STEP 2: BUDGET PLANNER Tools
// ============================================================================

// CreateBudgetTool creates an envelope-based budget.
func CreateBudgetTool(db *Database, executor core.ToolExecutor) core.Tool {
	return tools.New("create_budget").
		Description(`Create an envelope-based budget for the user. This tool:
1. Analyzes their income and spending patterns
2. Creates budget envelopes (Needs, Wants, Bills, Goals)
3. Sets up guardrails (soft alerts, hard stops)

Use default 50/30/20 rule or customize based on user preferences.`).
		Schema(tools.ObjectSchema(map[string]interface{}{
			"total_income":     tools.NumberProperty("Total monthly income to budget"),
			"needs_percent":    tools.NumberProperty("Percentage for needs (default: 50)"),
			"wants_percent":    tools.NumberProperty("Percentage for wants (default: 20)"),
			"bills_percent":    tools.NumberProperty("Percentage for bills (default: 20)"),
			"goals_percent":    tools.NumberProperty("Percentage for savings/goals (default: 10)"),
			"buffer":           tools.NumberProperty("Buffer amount to keep unallocated"),
			"start_date":       tools.IntegerProperty("Day of month budget starts (usually payday)"),
			"custom_envelopes": tools.StringProperty("JSON array of custom envelope objects (optional)"),
		}, "total_income")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				TotalIncome     float64 `json:"total_income"`
				NeedsPercent    float64 `json:"needs_percent"`
				WantsPercent    float64 `json:"wants_percent"`
				BillsPercent    float64 `json:"bills_percent"`
				GoalsPercent    float64 `json:"goals_percent"`
				Buffer          float64 `json:"buffer"`
				StartDate       int     `json:"start_date"`
				CustomEnvelopes string  `json:"custom_envelopes"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			userID := "user"

			// Set defaults (50/20/20/10 rule)
			if params.NeedsPercent == 0 {
				params.NeedsPercent = 50
			}
			if params.WantsPercent == 0 {
				params.WantsPercent = 20
			}
			if params.BillsPercent == 0 {
				params.BillsPercent = 20
			}
			if params.GoalsPercent == 0 {
				params.GoalsPercent = 10
			}
			if params.StartDate == 0 {
				params.StartDate = 1
			}

			// Calculate envelope amounts
			envelopes := []Envelope{
				{
					Name:      "Needs",
					Amount:    params.TotalIncome * params.NeedsPercent / 100,
					Guardrail: "hard",
					Threshold: 1.0,
					Category:  "needs",
					Color:     "#007AFF",
				},
				{
					Name:      "Wants",
					Amount:    params.TotalIncome * params.WantsPercent / 100,
					Guardrail: "soft",
					Threshold: 0.8,
					Category:  "wants",
					Color:     "#AF52DE",
				},
				{
					Name:      "Bills",
					Amount:    params.TotalIncome * params.BillsPercent / 100,
					Guardrail: "auto_pay",
					Threshold: 1.0,
					Category:  "bills",
					Color:     "#FF9500",
				},
				{
					Name:      "Goals",
					Amount:    params.TotalIncome * params.GoalsPercent / 100,
					Guardrail: "protected",
					Threshold: 1.0,
					Category:  "savings",
					Color:     "#34C759",
				},
			}

			budget := &Budget{
				UserID:      userID,
				Period:      "monthly",
				StartDate:   params.StartDate,
				Envelopes:   envelopes,
				Buffer:      params.Buffer,
				TotalBudget: params.TotalIncome,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := db.SaveBudget(budget); err != nil {
				return nil, fmt.Errorf("failed to save budget: %w", err)
			}

			// Format response
			envelopeSummary := make([]map[string]interface{}, len(envelopes))
			for i, e := range envelopes {
				envelopeSummary[i] = map[string]interface{}{
					"name":      e.Name,
					"amount":    fmt.Sprintf("£%.2f", e.Amount),
					"percent":   fmt.Sprintf("%.0f%%", (e.Amount/params.TotalIncome)*100),
					"guardrail": e.Guardrail,
				}
			}

			return map[string]interface{}{
				"success":      true,
				"message":      "Budget created successfully!",
				"total_budget": fmt.Sprintf("£%.2f", params.TotalIncome),
				"buffer":       fmt.Sprintf("£%.2f", params.Buffer),
				"envelopes":    envelopeSummary,
				"start_date":   params.StartDate,
				"next_step":    "Review your subscriptions to find potential savings.",
			}, nil
		}).
		Build()
}

// GetBudgetStatusTool gets current budget status with spending.
func GetBudgetStatusTool(db *Database, executor core.ToolExecutor) core.Tool {
	return tools.New("get_budget_status").
		Description("Get the current budget status showing spending against each envelope. Includes alerts if approaching limits.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			budget, err := db.GetBudget(userID)
			if err != nil {
				return nil, err
			}

			if budget == nil {
				return map[string]interface{}{
					"success": false,
					"message": "No budget found. Would you like to create one?",
				}, nil
			}

			// Calculate days remaining in budget period
			now := time.Now()
			daysInMonth := 30 // Simplified
			dayOfMonth := now.Day()
			daysRemaining := daysInMonth - dayOfMonth + budget.StartDate
			if daysRemaining > daysInMonth {
				daysRemaining -= daysInMonth
			}

			// Build status for each envelope
			envelopeStatus := make([]map[string]interface{}, len(budget.Envelopes))
			alerts := []string{}

			for i, e := range budget.Envelopes {
				percentUsed := 0.0
				if e.Amount > 0 {
					percentUsed = e.Spent / e.Amount
				}
				remaining := e.Amount - e.Spent
				onTrack := percentUsed <= (float64(daysInMonth-daysRemaining) / float64(daysInMonth))

				status := map[string]interface{}{
					"name":         e.Name,
					"allocated":    fmt.Sprintf("£%.2f", e.Amount),
					"spent":        fmt.Sprintf("£%.2f", e.Spent),
					"remaining":    fmt.Sprintf("£%.2f", remaining),
					"percent_used": fmt.Sprintf("%.0f%%", percentUsed*100),
					"on_track":     onTrack,
				}

				// Check guardrails
				if percentUsed >= e.Threshold && e.Guardrail == "soft" {
					alert := fmt.Sprintf("⚠️ %s is at %.0f%% - approaching limit", e.Name, percentUsed*100)
					alerts = append(alerts, alert)
					status["alert"] = alert
				}
				if percentUsed >= 1.0 && e.Guardrail == "hard" {
					alert := fmt.Sprintf("🛑 %s has reached its limit!", e.Name)
					alerts = append(alerts, alert)
					status["alert"] = alert
				}

				envelopeStatus[i] = status
			}

			return map[string]interface{}{
				"success":        true,
				"period":         budget.Period,
				"days_remaining": daysRemaining,
				"total_budget":   fmt.Sprintf("£%.2f", budget.TotalBudget),
				"envelopes":      envelopeStatus,
				"alerts":         alerts,
			}, nil
		}).
		Build()
}

// ============================================================================
// STEP 4: SMART SAVINGS Tools
// ============================================================================

// CreateSavingsRuleTool creates an automated savings rule.
func CreateSavingsRuleTool(db *Database) core.Tool {
	return tools.New("create_savings_rule").
		Description(`Create an automated savings rule. Available rule types:
- Payday sweep: Move fixed amount after income detected
- Round-ups: Save spare change from purchases
- Under-budget sweep: Move percentage of unspent budget
- Scheduled: Regular transfers on specific days`).
		Schema(tools.ObjectSchema(map[string]interface{}{
			"name":        tools.StringProperty("Name for this rule (e.g., 'Payday Sweep')"),
			"type":        tools.StringEnumProperty("Rule type", "scheduled", "per_transaction", "conditional"),
			"trigger":     tools.StringEnumProperty("What triggers the rule", "income_detected", "purchase", "budget_under", "weekly", "monthly"),
			"action":      tools.StringEnumProperty("What action to take", "transfer", "round_and_save", "transfer_percentage"),
			"amount":      tools.NumberProperty("Fixed amount to transfer (for 'transfer' action)"),
			"percentage":  tools.NumberProperty("Percentage to transfer (for 'transfer_percentage' action, e.g., 0.3 for 30%)"),
			"destination": tools.StringEnumProperty("Where to send the money", "savings_emergency", "savings_goals", "investment"),
		}, "name", "type", "trigger", "action", "destination")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Name        string  `json:"name"`
				Type        string  `json:"type"`
				Trigger     string  `json:"trigger"`
				Action      string  `json:"action"`
				Amount      float64 `json:"amount"`
				Percentage  float64 `json:"percentage"`
				Destination string  `json:"destination"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			userID := "user"

			rule := &SavingsRule{
				ID:          uuid.New().String(),
				UserID:      userID,
				Name:        params.Name,
				Type:        params.Type,
				Trigger:     params.Trigger,
				Action:      params.Action,
				Amount:      params.Amount,
				Percentage:  params.Percentage,
				Destination: params.Destination,
				Active:      true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := db.SaveSavingsRule(rule); err != nil {
				return nil, fmt.Errorf("failed to save rule: %w", err)
			}

			description := ""
			switch params.Action {
			case "transfer":
				description = fmt.Sprintf("Transfer £%.2f to %s when %s", params.Amount, params.Destination, params.Trigger)
			case "round_and_save":
				description = fmt.Sprintf("Round up purchases and save to %s", params.Destination)
			case "transfer_percentage":
				description = fmt.Sprintf("Transfer %.0f%% to %s when %s", params.Percentage*100, params.Destination, params.Trigger)
			}

			return map[string]interface{}{
				"success":     true,
				"message":     "Savings rule created!",
				"rule_id":     rule.ID,
				"name":        params.Name,
				"description": description,
				"active":      true,
			}, nil
		}).
		Build()
}

// GetSavingsRulesTool lists all savings rules.
func GetSavingsRulesTool(db *Database) core.Tool {
	return tools.New("get_savings_rules").
		Description("Get all savings rules for the user, including their status and total saved.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			rules, err := db.GetSavingsRules(userID)
			if err != nil {
				return nil, err
			}

			if len(rules) == 0 {
				return map[string]interface{}{
					"success": true,
					"rules":   []interface{}{},
					"message": "No savings rules set up yet. Would you like to create one?",
					"suggestions": []string{
						"Payday sweep: Automatically move money to savings after payday",
						"Round-ups: Save spare change from every purchase",
						"Under-budget sweep: Save a portion of unspent budget each week",
					},
				}, nil
			}

			totalSaved := 0.0
			rulesList := make([]map[string]interface{}, len(rules))
			for i, r := range rules {
				totalSaved += r.TotalSaved
				rulesList[i] = map[string]interface{}{
					"id":          r.ID,
					"name":        r.Name,
					"type":        r.Type,
					"trigger":     r.Trigger,
					"action":      r.Action,
					"amount":      r.Amount,
					"percentage":  r.Percentage,
					"destination": r.Destination,
					"active":      r.Active,
					"total_saved": fmt.Sprintf("£%.2f", r.TotalSaved),
					"run_count":   r.RunCount,
				}
			}

			return map[string]interface{}{
				"success":     true,
				"rules":       rulesList,
				"total_rules": len(rules),
				"total_saved": fmt.Sprintf("£%.2f", totalSaved),
			}, nil
		}).
		Build()
}

// GetEmergencyFundStatusTool gets emergency fund ladder progress.
func GetEmergencyFundStatusTool(db *Database) core.Tool {
	return tools.New("get_emergency_fund_status").
		Description("Get the user's emergency fund progress through the 3-stage ladder (2 weeks → 1 month → 3 months).").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			ef, err := db.GetEmergencyFund(userID)
			if err != nil {
				return nil, err
			}

			if ef == nil {
				return map[string]interface{}{
					"success": false,
					"message": "Emergency fund not set up. Complete onboarding first to set your targets.",
				}, nil
			}

			status := ef.GetStatus()
			status["success"] = true

			// Add investing eligibility message
			if ef.CurrentStage >= 1 {
				status["investing_unlocked"] = true
				status["message"] = "Stage 1 complete! You can now start investing your surplus."
			} else {
				status["investing_unlocked"] = false
				remaining := ef.Stage1Target - ef.Current
				status["message"] = fmt.Sprintf("£%.2f more to unlock investing (Stage 1)", remaining)
			}

			return status, nil
		}).
		Build()
}

// ============================================================================
// STEP 5: INVESTMENT SURPLUS Tools
// ============================================================================

// GetInvestmentSurplusTool gets the safe amount available to invest.
func GetInvestmentSurplusTool(db *Database) core.Tool {
	return tools.New("get_investment_surplus").
		Description("Get the amount available to invest. This is only unlocked after completing Stage 1 of the emergency fund.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			// Check emergency fund status first
			ef, _ := db.GetEmergencyFund(userID)
			if ef == nil || ef.CurrentStage < 1 {
				return map[string]interface{}{
					"success":   false,
					"available": 0,
					"locked":    true,
					"message":   "Investing is locked until you complete Stage 1 of your emergency fund.",
					"unlock_at": ef.Stage1Target,
					"current":   ef.Current,
					"remaining": ef.Stage1Target - ef.Current,
				}, nil
			}

			surplus, _ := db.GetInvestmentSurplus(userID)
			if surplus == nil {
				surplus = &InvestmentSurplus{
					UserID:            userID,
					Available:         0,
					CoreAllocation:    0.8,
					ExploreAllocation: 0.2,
					DCAFrequency:      "weekly",
				}
			}

			coreAmount := surplus.Available * surplus.CoreAllocation
			exploreAmount := surplus.Available * surplus.ExploreAllocation

			return map[string]interface{}{
				"success":            true,
				"locked":             false,
				"available":          fmt.Sprintf("£%.2f", surplus.Available),
				"core_allocation":    fmt.Sprintf("£%.2f (%.0f%%)", coreAmount, surplus.CoreAllocation*100),
				"explore_allocation": fmt.Sprintf("£%.2f (%.0f%%)", exploreAmount, surplus.ExploreAllocation*100),
				"pending_dca":        fmt.Sprintf("£%.2f", surplus.PendingDCA),
				"dca_frequency":      surplus.DCAFrequency,
				"next_dca_date":      surplus.NextDCADate,
				"message":            "Your surplus is ready to invest!",
			}, nil
		}).
		Build()
}

// ============================================================================
// STEP 6: AUTOPILOT Tools
// ============================================================================

// GetWeeklyDigestTool generates the weekly summary.
func GetWeeklyDigestTool(db *Database, executor core.ToolExecutor) core.Tool {
	return tools.New("get_weekly_digest").
		Description("Generate the weekly financial digest showing budget status, savings progress, and pending actions to approve.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			userID := "user"

			// Get budget status
			budget, _ := db.GetBudget(userID)

			// Get pending approvals
			approvals, _ := db.GetPendingApprovals(userID)

			// Get emergency fund
			ef, _ := db.GetEmergencyFund(userID)

			// Get savings rules summary
			rules, _ := db.GetSavingsRules(userID)

			now := time.Now()
			weekStart := now.AddDate(0, 0, -int(now.Weekday()))
			weekEnd := weekStart.AddDate(0, 0, 6)

			digest := map[string]interface{}{
				"success":    true,
				"week_start": weekStart.Format("Jan 2"),
				"week_end":   weekEnd.Format("Jan 2"),
			}

			// Budget summary
			if budget != nil {
				budgetSummary := make([]map[string]string, len(budget.Envelopes))
				for i, e := range budget.Envelopes {
					pct := 0.0
					if e.Amount > 0 {
						pct = e.Spent / e.Amount * 100
					}
					budgetSummary[i] = map[string]string{
						"name":   e.Name,
						"status": fmt.Sprintf("%.0f%% used", pct),
						"amount": fmt.Sprintf("£%.2f of £%.2f", e.Spent, e.Amount),
					}
				}
				digest["budget_summary"] = budgetSummary
			}

			// Emergency fund progress
			if ef != nil {
				digest["emergency_fund"] = map[string]interface{}{
					"current":  fmt.Sprintf("£%.2f", ef.Current),
					"stage":    ef.CurrentStage,
					"progress": fmt.Sprintf("%.0f%%", (ef.Current/ef.Stage2Target)*100),
				}
			}

			// Savings rules
			if len(rules) > 0 {
				totalSaved := 0.0
				for _, r := range rules {
					totalSaved += r.TotalSaved
				}
				digest["savings_this_week"] = fmt.Sprintf("£%.2f", totalSaved)
				digest["active_rules"] = len(rules)
			}

			// Pending actions
			if len(approvals) > 0 {
				pendingList := make([]map[string]string, len(approvals))
				for i, a := range approvals {
					pendingList[i] = map[string]string{
						"id":          a.ID,
						"type":        a.Type,
						"description": a.Description,
						"amount":      fmt.Sprintf("£%.2f", a.Amount),
					}
				}
				digest["pending_actions"] = pendingList
				digest["actions_needed"] = len(approvals)
			} else {
				digest["pending_actions"] = []interface{}{}
				digest["actions_needed"] = 0
			}

			return digest, nil
		}).
		Build()
}

// ApprovePendingActionTool approves a pending action.
func ApprovePendingActionTool(db *Database) core.Tool {
	return tools.New("approve_pending_action").
		Description("Approve a pending action from the weekly digest (e.g., sweep to savings, cancel subscription).").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"action_id": tools.StringProperty("ID of the pending action to approve"),
		}, "action_id")).
		RequiresConfirmation().
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				ActionID string `json:"action_id"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			if err := db.ApprovePendingAction(params.ActionID); err != nil {
				return nil, fmt.Errorf("failed to approve action: %w", err)
			}

			return map[string]interface{}{
				"success": true,
				"message": "Action approved and will be executed.",
			}, nil
		}).
		Build()
}

// ============================================================================
// HELPER: Get all journey tools
// ============================================================================

// GetJourneyTools returns all tools for the journey flow.
func GetJourneyTools(db *Database, executor core.ToolExecutor) []core.Tool {
	return []core.Tool{
		// Step 1: Chat with Nim
		CreateUserPlanTool(db),
		GetUserPlanTool(db),
		GetJourneyStatusTool(db),

		// Step 2: Budget Planner
		CreateBudgetTool(db, executor),
		GetBudgetStatusTool(db, executor),

		// Step 4: Smart Savings
		CreateSavingsRuleTool(db),
		GetSavingsRulesTool(db),
		GetEmergencyFundStatusTool(db),

		// Step 5: Investment Surplus
		GetInvestmentSurplusTool(db),

		// Step 6: Autopilot
		GetWeeklyDigestTool(db, executor),
		ApprovePendingActionTool(db),
	}
}
