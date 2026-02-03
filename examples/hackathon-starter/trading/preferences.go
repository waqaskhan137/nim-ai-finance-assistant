package trading

import (
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// TRADING PREFERENCES & USER CONFIGURATION
// ============================================================================

// TradingPreferences stores user trading preferences and settings.
type TradingPreferences struct {
	UserID          string    `json:"user_id"`
	Assets          []string  `json:"assets"`           // ["BTCUSDT", "ETHUSDT"]
	Style           string    `json:"style"`            // "day_trading", "swing", "hold"
	RiskProfile     string    `json:"risk_profile"`     // "conservative", "moderate", "aggressive"
	ProfitTarget    float64   `json:"profit_target"`    // e.g., 10% gain target
	MaxLossPercent  float64   `json:"max_loss_percent"` // e.g., 5% max loss
	AutoTrade       bool      `json:"auto_trade"`       // Enable autonomous trading
	TradingHours    *TimeRange `json:"trading_hours,omitempty"` // Optional: only trade during certain hours
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TimeRange defines trading hours.
type TimeRange struct {
	Start time.Time `json:"start"` // e.g., "09:00"
	End   time.Time `json:"end"`   // e.g., "17:00"
}

// TradingAllocation represents a user's allocated trading budget.
type TradingAllocation struct {
	UserID          string    `json:"user_id"`
	Amount          float64   `json:"amount"`           // Allocated amount in USD
	StopLossFloor   float64   `json:"stop_loss_floor"`  // Minimum portfolio value
	RiskProfile     string    `json:"risk_profile"`     // Risk profile used
	BinanceAccount  string    `json:"binance_account"`  // Binance account ID (if applicable)
	Status          string    `json:"status"`           // "active", "closed", "suspended"
	AllocatedAt     time.Time `json:"allocated_at"`
	ClosedAt        *time.Time `json:"closed_at,omitempty"`
	FinalValue      *float64  `json:"final_value,omitempty"`
	TotalPnL        *float64  `json:"total_pnl,omitempty"`
}

// DefaultTradingPreferences returns sensible defaults.
func DefaultTradingPreferences(userID string) *TradingPreferences {
	return &TradingPreferences{
		UserID:         userID,
		Assets:         []string{"BTCUSDT", "ETHUSDT"},
		Style:          "swing",
		RiskProfile:    "moderate",
		ProfitTarget:   0.10, // 10% profit target
		MaxLossPercent: 0.05, // 5% max loss
		AutoTrade:      false, // Start with manual control
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// Validate checks if preferences are valid.
func (tp *TradingPreferences) Validate() error {
	if tp.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	validStyles := map[string]bool{
		"day_trading": true,
		"swing":       true,
		"hold":        true,
	}
	if !validStyles[tp.Style] {
		return fmt.Errorf("invalid style: %s (must be day_trading, swing, or hold)", tp.Style)
	}

	validRiskProfiles := map[string]bool{
		"conservative": true,
		"moderate":     true,
		"aggressive":   true,
	}
	if !validRiskProfiles[tp.RiskProfile] {
		return fmt.Errorf("invalid risk_profile: %s (must be conservative, moderate, or aggressive)", tp.RiskProfile)
	}

	if len(tp.Assets) == 0 {
		return fmt.Errorf("at least one asset must be specified")
	}

	if tp.ProfitTarget <= 0 || tp.ProfitTarget > 1 {
		return fmt.Errorf("profit_target must be between 0 and 1 (e.g., 0.10 for 10%%)")
	}

	if tp.MaxLossPercent <= 0 || tp.MaxLossPercent > 0.5 {
		return fmt.Errorf("max_loss_percent must be between 0 and 0.5 (e.g., 0.05 for 5%%)")
	}

	return nil
}

// ToJSON converts preferences to JSON string.
func (tp *TradingPreferences) ToJSON() (string, error) {
	data, err := json.Marshal(tp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON loads preferences from JSON string.
func (tp *TradingPreferences) FromJSON(jsonStr string) error {
	return json.Unmarshal([]byte(jsonStr), tp)
}