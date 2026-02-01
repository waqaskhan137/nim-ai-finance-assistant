// Package trading - Journey models for the Stabilize → Save → Invest flow
package trading

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ============================================================================
// USER PLAN - Step 1: Chat with Nim
// ============================================================================

// UserPlan represents the user's financial goals and rules created during onboarding.
type UserPlan struct {
	UserID             string         `json:"user_id"`
	Goals              Goals          `json:"goals"`
	Rules              FinancialRules `json:"rules"`
	Income             IncomeProfile  `json:"income"`
	ConnectedAccounts  []string       `json:"connected_accounts"`
	OnboardingComplete bool           `json:"onboarding_complete"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// Goals represents the user's financial goals.
type Goals struct {
	MonthlySavingsTarget float64  `json:"monthly_savings_target"`
	EmergencyFundTarget  float64  `json:"emergency_fund_target"`
	InvestmentGoal       string   `json:"investment_goal"` // "grow_wealth", "retirement", "big_purchase"
	ShortTermGoals       []string `json:"short_term_goals"`
}

// FinancialRules represents spending and saving rules.
type FinancialRules struct {
	BufferAmount        float64 `json:"buffer_amount"`
	DiningCap           float64 `json:"dining_cap,omitempty"`
	EntertainmentCap    float64 `json:"entertainment_cap,omitempty"`
	InvestmentFrequency string  `json:"investment_frequency"` // "weekly", "biweekly", "monthly"
	RiskProfile         string  `json:"risk_profile"`         // "conservative", "moderate", "aggressive"
	AutoSweepEnabled    bool    `json:"auto_sweep_enabled"`
	RoundUpsEnabled     bool    `json:"round_ups_enabled"`
}

// IncomeProfile represents the user's income pattern.
type IncomeProfile struct {
	Cadence         string  `json:"cadence"`                 // "weekly", "biweekly", "monthly", "irregular"
	ExpectedDate    int     `json:"expected_date,omitempty"` // Day of month (1-31)
	EstimatedAmount float64 `json:"estimated_amount"`
}

// ============================================================================
// BUDGET - Step 2: Budget Planner
// ============================================================================

// Budget represents an envelope-based budget aligned to cashflow.
type Budget struct {
	UserID      string     `json:"user_id"`
	Period      string     `json:"period"`     // "monthly", "weekly"
	StartDate   int        `json:"start_date"` // Day of month budget starts (usually payday)
	Envelopes   []Envelope `json:"envelopes"`
	Buffer      float64    `json:"buffer"`
	TotalBudget float64    `json:"total_budget"`
	Insights    []string   `json:"insights,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Envelope represents a budget category with guardrails.
type Envelope struct {
	Name      string  `json:"name"`
	Amount    float64 `json:"amount"`
	Spent     float64 `json:"spent"`
	Guardrail string  `json:"guardrail"` // "soft", "hard", "auto_pay", "protected"
	Threshold float64 `json:"threshold"` // Alert threshold (e.g., 0.8 for 80%)
	Category  string  `json:"category"`  // Spending category for auto-categorization
	Color     string  `json:"color,omitempty"`
}

// BudgetStatus represents current spending against budget.
type BudgetStatus struct {
	Envelope      string  `json:"envelope"`
	Allocated     float64 `json:"allocated"`
	Spent         float64 `json:"spent"`
	Remaining     float64 `json:"remaining"`
	PercentUsed   float64 `json:"percent_used"`
	DaysRemaining int     `json:"days_remaining"`
	OnTrack       bool    `json:"on_track"`
	Alert         string  `json:"alert,omitempty"`
}

// ============================================================================
// SAVINGS RULES - Step 4: Smart Savings
// ============================================================================

// SavingsRule represents an automated savings rule.
type SavingsRule struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`    // "scheduled", "per_transaction", "conditional"
	Trigger     string     `json:"trigger"` // "income_detected", "purchase", "budget_under", "manual"
	Action      string     `json:"action"`  // "transfer", "round_and_save", "transfer_percentage"
	Amount      float64    `json:"amount,omitempty"`
	Percentage  float64    `json:"percentage,omitempty"`
	Destination string     `json:"destination"` // "savings_emergency", "savings_goals", "investment"
	Active      bool       `json:"active"`
	LastRun     *time.Time `json:"last_run,omitempty"`
	TotalSaved  float64    `json:"total_saved"`
	RunCount    int        `json:"run_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// EmergencyFund represents the emergency fund ladder progress.
type EmergencyFund struct {
	UserID          string    `json:"user_id"`
	Current         float64   `json:"current"`
	Stage1Target    float64   `json:"stage_1_target"` // 2 weeks expenses
	Stage2Target    float64   `json:"stage_2_target"` // 1 month expenses
	Stage3Target    float64   `json:"stage_3_target"` // 3-6 months expenses
	CurrentStage    int       `json:"current_stage"`  // 0, 1, 2, or 3
	MonthlyExpenses float64   `json:"monthly_expenses"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// EmergencyFundStatus returns the current stage progress.
func (ef *EmergencyFund) GetStatus() map[string]interface{} {
	stage1Progress := min(ef.Current/ef.Stage1Target, 1.0)
	stage2Progress := 0.0
	stage3Progress := 0.0

	if ef.Current >= ef.Stage1Target {
		stage2Progress = min((ef.Current-ef.Stage1Target)/(ef.Stage2Target-ef.Stage1Target), 1.0)
	}
	if ef.Current >= ef.Stage2Target {
		stage3Progress = min((ef.Current-ef.Stage2Target)/(ef.Stage3Target-ef.Stage2Target), 1.0)
	}

	return map[string]interface{}{
		"current_amount":   ef.Current,
		"current_stage":    ef.CurrentStage,
		"stage_1_target":   ef.Stage1Target,
		"stage_1_progress": stage1Progress,
		"stage_1_complete": ef.Current >= ef.Stage1Target,
		"stage_2_target":   ef.Stage2Target,
		"stage_2_progress": stage2Progress,
		"stage_2_complete": ef.Current >= ef.Stage2Target,
		"stage_3_target":   ef.Stage3Target,
		"stage_3_progress": stage3Progress,
		"stage_3_complete": ef.Current >= ef.Stage3Target,
		"ready_to_invest":  ef.CurrentStage >= 1,
	}
}

// ============================================================================
// INVESTMENT SURPLUS - Step 5: Trading Terminal
// ============================================================================

// InvestmentSurplus represents the safe amount available to invest.
type InvestmentSurplus struct {
	UserID            string     `json:"user_id"`
	Available         float64    `json:"available"`
	Source            string     `json:"source"`             // "weekly_sweep", "manual", "under_budget"
	CoreAllocation    float64    `json:"core_allocation"`    // 80% default
	ExploreAllocation float64    `json:"explore_allocation"` // 20% default
	PendingDCA        float64    `json:"pending_dca"`
	DCAFrequency      string     `json:"dca_frequency"`
	LastDCADate       *time.Time `json:"last_dca_date,omitempty"`
	NextDCADate       *time.Time `json:"next_dca_date,omitempty"`
}

// ============================================================================
// AUTOPILOT - Step 6
// ============================================================================

// PendingAction represents an action awaiting user approval.
type PendingApproval struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Type        string     `json:"type"` // "sweep", "subscription_cancel", "rebalance", "dca"
	Description string     `json:"description"`
	Amount      float64    `json:"amount,omitempty"`
	Details     string     `json:"details,omitempty"` // JSON blob
	Status      string     `json:"status"`            // "pending", "approved", "rejected", "expired"
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// WeeklyDigest represents the weekly summary.
type WeeklyDigest struct {
	UserID                string                  `json:"user_id"`
	WeekStart             time.Time               `json:"week_start"`
	WeekEnd               time.Time               `json:"week_end"`
	BudgetStatus          map[string]BudgetStatus `json:"budget_status"`
	TotalSpent            float64                 `json:"total_spent"`
	TotalSaved            float64                 `json:"total_saved"`
	SavingsRulesRun       int                     `json:"savings_rules_run"`
	EmergencyFundProgress float64                 `json:"emergency_fund_progress"`
	DCAExecuted           float64                 `json:"dca_executed,omitempty"`
	PendingActions        []PendingApproval       `json:"pending_actions"`
	Alerts                []string                `json:"alerts,omitempty"`
	Highlights            []string                `json:"highlights,omitempty"`
}

// ============================================================================
// SUBSCRIPTION ENHANCEMENTS - Step 3
// ============================================================================

// SubscriptionAction represents a user action on a subscription.
type SubscriptionAction struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	Merchant       string     `json:"merchant"`
	Action         string     `json:"action"` // "keep", "pause", "cancel", "negotiate"
	PreviousAmount float64    `json:"previous_amount"`
	Notes          string     `json:"notes,omitempty"`
	ReminderDate   *time.Time `json:"reminder_date,omitempty"` // For "pause" actions
	CreatedAt      time.Time  `json:"created_at"`
}

// ============================================================================
// DATABASE SCHEMA EXTENSION
// ============================================================================

// InitializeJourneySchema adds the journey tables to the database.
func (d *Database) InitializeJourneySchema() error {
	schema := `
	-- User Plans (Step 1: Chat with Nim)
	CREATE TABLE IF NOT EXISTS user_plans (
		user_id TEXT PRIMARY KEY,
		goals TEXT NOT NULL,           -- JSON
		rules TEXT NOT NULL,           -- JSON
		income TEXT NOT NULL,          -- JSON
		connected_accounts TEXT,       -- JSON array
		onboarding_complete BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Budgets (Step 2: Budget Planner)
	CREATE TABLE IF NOT EXISTS budgets (
		user_id TEXT PRIMARY KEY,
		period TEXT NOT NULL,
		start_date INTEGER NOT NULL,
		envelopes TEXT NOT NULL,       -- JSON array
		buffer REAL NOT NULL,
		total_budget REAL NOT NULL,
		insights TEXT,                 -- JSON array
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Savings Rules (Step 4: Smart Savings)
	CREATE TABLE IF NOT EXISTS savings_rules (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		trigger TEXT NOT NULL,
		action TEXT NOT NULL,
		amount REAL,
		percentage REAL,
		destination TEXT NOT NULL,
		active BOOLEAN DEFAULT 1,
		last_run DATETIME,
		total_saved REAL DEFAULT 0,
		run_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Emergency Fund (Step 4: Smart Savings)
	CREATE TABLE IF NOT EXISTS emergency_funds (
		user_id TEXT PRIMARY KEY,
		current REAL DEFAULT 0,
		stage_1_target REAL NOT NULL,
		stage_2_target REAL NOT NULL,
		stage_3_target REAL NOT NULL,
		current_stage INTEGER DEFAULT 0,
		monthly_expenses REAL NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Investment Surplus (Step 5: Trading)
	CREATE TABLE IF NOT EXISTS investment_surplus (
		user_id TEXT PRIMARY KEY,
		available REAL DEFAULT 0,
		source TEXT,
		core_allocation REAL DEFAULT 0.8,
		explore_allocation REAL DEFAULT 0.2,
		pending_dca REAL DEFAULT 0,
		dca_frequency TEXT DEFAULT 'weekly',
		last_dca_date DATETIME,
		next_dca_date DATETIME
	);

	-- Pending Approvals (Step 6: Autopilot)
	CREATE TABLE IF NOT EXISTS pending_approvals (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		description TEXT NOT NULL,
		amount REAL,
		details TEXT,
		status TEXT DEFAULT 'pending',
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		resolved_at DATETIME
	);

	-- Subscription Actions (Step 3: Subscriptions)
	CREATE TABLE IF NOT EXISTS subscription_actions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		merchant TEXT NOT NULL,
		action TEXT NOT NULL,
		previous_amount REAL,
		notes TEXT,
		reminder_date DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_savings_rules_user ON savings_rules(user_id);
	CREATE INDEX IF NOT EXISTS idx_pending_approvals_user ON pending_approvals(user_id);
	CREATE INDEX IF NOT EXISTS idx_pending_approvals_status ON pending_approvals(status);
	CREATE INDEX IF NOT EXISTS idx_subscription_actions_user ON subscription_actions(user_id);
	`

	_, err := d.db.Exec(schema)
	return err
}

// ============================================================================
// USER PLAN CRUD
// ============================================================================

// SaveUserPlan saves or updates a user plan.
func (d *Database) SaveUserPlan(plan *UserPlan) error {
	goalsJSON, _ := json.Marshal(plan.Goals)
	rulesJSON, _ := json.Marshal(plan.Rules)
	incomeJSON, _ := json.Marshal(plan.Income)
	accountsJSON, _ := json.Marshal(plan.ConnectedAccounts)

	query := `
		INSERT OR REPLACE INTO user_plans
		(user_id, goals, rules, income, connected_accounts, onboarding_complete, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM user_plans WHERE user_id = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)
	`

	_, err := d.db.Exec(query,
		plan.UserID,
		string(goalsJSON),
		string(rulesJSON),
		string(incomeJSON),
		string(accountsJSON),
		plan.OnboardingComplete,
		plan.UserID,
	)
	return err
}

// GetUserPlan retrieves a user plan.
func (d *Database) GetUserPlan(userID string) (*UserPlan, error) {
	query := `
		SELECT user_id, goals, rules, income, connected_accounts, onboarding_complete, created_at, updated_at
		FROM user_plans WHERE user_id = ?
	`

	var plan UserPlan
	var goalsJSON, rulesJSON, incomeJSON, accountsJSON string

	err := d.db.QueryRow(query, userID).Scan(
		&plan.UserID,
		&goalsJSON,
		&rulesJSON,
		&incomeJSON,
		&accountsJSON,
		&plan.OnboardingComplete,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	json.Unmarshal([]byte(goalsJSON), &plan.Goals)
	json.Unmarshal([]byte(rulesJSON), &plan.Rules)
	json.Unmarshal([]byte(incomeJSON), &plan.Income)
	json.Unmarshal([]byte(accountsJSON), &plan.ConnectedAccounts)

	return &plan, nil
}

// ============================================================================
// BUDGET CRUD
// ============================================================================

// SaveBudget saves or updates a budget.
func (d *Database) SaveBudget(budget *Budget) error {
	envelopesJSON, _ := json.Marshal(budget.Envelopes)
	insightsJSON, _ := json.Marshal(budget.Insights)

	query := `
		INSERT OR REPLACE INTO budgets
		(user_id, period, start_date, envelopes, buffer, total_budget, insights, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM budgets WHERE user_id = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)
	`

	_, err := d.db.Exec(query,
		budget.UserID,
		budget.Period,
		budget.StartDate,
		string(envelopesJSON),
		budget.Buffer,
		budget.TotalBudget,
		string(insightsJSON),
		budget.UserID,
	)
	return err
}

// GetBudget retrieves a budget.
func (d *Database) GetBudget(userID string) (*Budget, error) {
	query := `
		SELECT user_id, period, start_date, envelopes, buffer, total_budget, insights, created_at, updated_at
		FROM budgets WHERE user_id = ?
	`

	var budget Budget
	var envelopesJSON, insightsJSON string

	err := d.db.QueryRow(query, userID).Scan(
		&budget.UserID,
		&budget.Period,
		&budget.StartDate,
		&envelopesJSON,
		&budget.Buffer,
		&budget.TotalBudget,
		&insightsJSON,
		&budget.CreatedAt,
		&budget.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	json.Unmarshal([]byte(envelopesJSON), &budget.Envelopes)
	json.Unmarshal([]byte(insightsJSON), &budget.Insights)

	return &budget, nil
}

// ============================================================================
// SAVINGS RULES CRUD
// ============================================================================

// SaveSavingsRule saves or updates a savings rule.
func (d *Database) SaveSavingsRule(rule *SavingsRule) error {
	query := `
		INSERT OR REPLACE INTO savings_rules
		(id, user_id, name, type, trigger, action, amount, percentage, destination, active, last_run, total_saved, run_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM savings_rules WHERE id = ?), CURRENT_TIMESTAMP), CURRENT_TIMESTAMP)
	`

	_, err := d.db.Exec(query,
		rule.ID,
		rule.UserID,
		rule.Name,
		rule.Type,
		rule.Trigger,
		rule.Action,
		rule.Amount,
		rule.Percentage,
		rule.Destination,
		rule.Active,
		rule.LastRun,
		rule.TotalSaved,
		rule.RunCount,
		rule.ID,
	)
	return err
}

// GetSavingsRules retrieves all savings rules for a user.
func (d *Database) GetSavingsRules(userID string) ([]SavingsRule, error) {
	query := `
		SELECT id, user_id, name, type, trigger, action, amount, percentage, destination, active, last_run, total_saved, run_count, created_at, updated_at
		FROM savings_rules WHERE user_id = ? ORDER BY created_at
	`

	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []SavingsRule
	for rows.Next() {
		var rule SavingsRule
		var lastRun sql.NullTime

		err := rows.Scan(
			&rule.ID, &rule.UserID, &rule.Name, &rule.Type, &rule.Trigger, &rule.Action,
			&rule.Amount, &rule.Percentage, &rule.Destination, &rule.Active,
			&lastRun, &rule.TotalSaved, &rule.RunCount, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if lastRun.Valid {
			rule.LastRun = &lastRun.Time
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// ============================================================================
// EMERGENCY FUND CRUD
// ============================================================================

// SaveEmergencyFund saves or updates emergency fund status.
func (d *Database) SaveEmergencyFund(ef *EmergencyFund) error {
	// Calculate current stage
	ef.CurrentStage = 0
	if ef.Current >= ef.Stage1Target {
		ef.CurrentStage = 1
	}
	if ef.Current >= ef.Stage2Target {
		ef.CurrentStage = 2
	}
	if ef.Current >= ef.Stage3Target {
		ef.CurrentStage = 3
	}

	query := `
		INSERT OR REPLACE INTO emergency_funds
		(user_id, current, stage_1_target, stage_2_target, stage_3_target, current_stage, monthly_expenses, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := d.db.Exec(query,
		ef.UserID,
		ef.Current,
		ef.Stage1Target,
		ef.Stage2Target,
		ef.Stage3Target,
		ef.CurrentStage,
		ef.MonthlyExpenses,
	)
	return err
}

// GetEmergencyFund retrieves emergency fund status.
func (d *Database) GetEmergencyFund(userID string) (*EmergencyFund, error) {
	query := `
		SELECT user_id, current, stage_1_target, stage_2_target, stage_3_target, current_stage, monthly_expenses, updated_at
		FROM emergency_funds WHERE user_id = ?
	`

	var ef EmergencyFund
	err := d.db.QueryRow(query, userID).Scan(
		&ef.UserID,
		&ef.Current,
		&ef.Stage1Target,
		&ef.Stage2Target,
		&ef.Stage3Target,
		&ef.CurrentStage,
		&ef.MonthlyExpenses,
		&ef.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &ef, nil
}

// ============================================================================
// PENDING APPROVALS CRUD
// ============================================================================

// SavePendingApproval saves a pending approval.
func (d *Database) SavePendingApproval(pa *PendingApproval) error {
	query := `
		INSERT OR REPLACE INTO pending_approvals
		(id, user_id, type, description, amount, details, status, expires_at, created_at, resolved_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		pa.ID, pa.UserID, pa.Type, pa.Description, pa.Amount, pa.Details,
		pa.Status, pa.ExpiresAt, pa.CreatedAt, pa.ResolvedAt,
	)
	return err
}

// GetPendingApprovals retrieves pending approvals for a user.
func (d *Database) GetPendingApprovals(userID string) ([]PendingApproval, error) {
	query := `
		SELECT id, user_id, type, description, amount, details, status, expires_at, created_at, resolved_at
		FROM pending_approvals WHERE user_id = ? AND status = 'pending' AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at
	`

	rows, err := d.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []PendingApproval
	for rows.Next() {
		var pa PendingApproval
		var amount sql.NullFloat64
		var details sql.NullString
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&pa.ID, &pa.UserID, &pa.Type, &pa.Description, &amount, &details,
			&pa.Status, &pa.ExpiresAt, &pa.CreatedAt, &resolvedAt,
		)
		if err != nil {
			continue
		}
		if amount.Valid {
			pa.Amount = amount.Float64
		}
		if details.Valid {
			pa.Details = details.String
		}
		if resolvedAt.Valid {
			pa.ResolvedAt = &resolvedAt.Time
		}
		approvals = append(approvals, pa)
	}

	return approvals, nil
}

// ApprovePendingAction approves a pending action.
func (d *Database) ApprovePendingAction(id string) error {
	now := time.Now()
	query := `UPDATE pending_approvals SET status = 'approved', resolved_at = ? WHERE id = ?`
	_, err := d.db.Exec(query, now, id)
	return err
}

// RejectPendingAction rejects a pending action.
func (d *Database) RejectPendingAction(id string) error {
	now := time.Now()
	query := `UPDATE pending_approvals SET status = 'rejected', resolved_at = ? WHERE id = ?`
	_, err := d.db.Exec(query, now, id)
	return err
}

// ============================================================================
// INVESTMENT SURPLUS CRUD
// ============================================================================

// SaveInvestmentSurplus saves investment surplus.
func (d *Database) SaveInvestmentSurplus(is *InvestmentSurplus) error {
	query := `
		INSERT OR REPLACE INTO investment_surplus
		(user_id, available, source, core_allocation, explore_allocation, pending_dca, dca_frequency, last_dca_date, next_dca_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query,
		is.UserID, is.Available, is.Source, is.CoreAllocation, is.ExploreAllocation,
		is.PendingDCA, is.DCAFrequency, is.LastDCADate, is.NextDCADate,
	)
	return err
}

// GetInvestmentSurplus retrieves investment surplus.
func (d *Database) GetInvestmentSurplus(userID string) (*InvestmentSurplus, error) {
	query := `
		SELECT user_id, available, source, core_allocation, explore_allocation, pending_dca, dca_frequency, last_dca_date, next_dca_date
		FROM investment_surplus WHERE user_id = ?
	`

	var is InvestmentSurplus
	var lastDCA, nextDCA sql.NullTime
	var source sql.NullString

	err := d.db.QueryRow(query, userID).Scan(
		&is.UserID, &is.Available, &source, &is.CoreAllocation, &is.ExploreAllocation,
		&is.PendingDCA, &is.DCAFrequency, &lastDCA, &nextDCA,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if source.Valid {
		is.Source = source.String
	}
	if lastDCA.Valid {
		is.LastDCADate = &lastDCA.Time
	}
	if nextDCA.Valid {
		is.NextDCADate = &nextDCA.Time
	}

	return &is, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// CalculateEmergencyFundTargets calculates targets based on monthly expenses.
func CalculateEmergencyFundTargets(monthlyExpenses float64) (stage1, stage2, stage3 float64) {
	stage1 = monthlyExpenses * 0.5 // 2 weeks
	stage2 = monthlyExpenses       // 1 month
	stage3 = monthlyExpenses * 3   // 3 months
	return
}

// CreateDefaultEnvelopes creates default budget envelopes based on income.
func CreateDefaultEnvelopes(totalIncome float64) []Envelope {
	return []Envelope{
		{Name: "Needs", Amount: totalIncome * 0.50, Guardrail: "hard", Threshold: 1.0, Category: "needs", Color: "#007AFF"},
		{Name: "Wants", Amount: totalIncome * 0.20, Guardrail: "soft", Threshold: 0.8, Category: "wants", Color: "#AF52DE"},
		{Name: "Bills", Amount: totalIncome * 0.20, Guardrail: "auto_pay", Threshold: 1.0, Category: "bills", Color: "#FF9500"},
		{Name: "Goals", Amount: totalIncome * 0.10, Guardrail: "protected", Threshold: 1.0, Category: "savings", Color: "#34C759"},
	}
}

// Helper function
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GetJourneyStatus returns the user's progress through the journey.
func (d *Database) GetJourneyStatus(userID string) (map[string]interface{}, error) {
	plan, _ := d.GetUserPlan(userID)
	budget, _ := d.GetBudget(userID)
	ef, _ := d.GetEmergencyFund(userID)
	rules, _ := d.GetSavingsRules(userID)
	surplus, _ := d.GetInvestmentSurplus(userID)

	status := map[string]interface{}{
		"step_1_chat":          plan != nil && plan.OnboardingComplete,
		"step_2_budget":        budget != nil,
		"step_3_subscriptions": true, // Always accessible
		"step_4_savings":       len(rules) > 0,
		"step_5_investing":     ef != nil && ef.CurrentStage >= 1,
		"step_6_autopilot":     plan != nil && plan.OnboardingComplete,
	}

	if plan != nil {
		status["plan"] = plan
	}
	if budget != nil {
		status["budget"] = budget
	}
	if ef != nil {
		status["emergency_fund"] = ef.GetStatus()
	}
	if surplus != nil {
		status["investment_surplus"] = surplus
	}

	// Calculate next step
	if plan == nil || !plan.OnboardingComplete {
		status["next_step"] = "Complete onboarding with Nim"
		status["next_step_number"] = 1
	} else if budget == nil {
		status["next_step"] = "Create your budget"
		status["next_step_number"] = 2
	} else if len(rules) == 0 {
		status["next_step"] = "Set up savings rules"
		status["next_step_number"] = 4
	} else if ef == nil || ef.CurrentStage < 1 {
		status["next_step"] = "Build your emergency fund to Stage 1"
		status["next_step_number"] = 4
	} else {
		status["next_step"] = "You're ready to invest!"
		status["next_step_number"] = 5
	}

	return status, nil
}
