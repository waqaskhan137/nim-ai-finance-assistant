package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// ============================================================================
// BUDGET & PORTFOLIO MANAGEMENT
// ============================================================================

// Position represents an open trading position.
type Position struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"` // "long" or "short"
	EntryPrice float64   `json:"entry_price"`
	Quantity   float64   `json:"quantity"`
	OpenTime   time.Time `json:"open_time"`
	StopLoss   float64   `json:"stop_loss,omitempty"`
	TakeProfit float64   `json:"take_profit,omitempty"`
}

// Trade represents a completed trade.
type Trade struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Quantity   float64   `json:"quantity"`
	PnL        float64   `json:"pnl"`
	OpenTime   time.Time `json:"open_time"`
	CloseTime  time.Time `json:"close_time"`
}

// RiskProfile defines trading risk parameters.
type RiskProfile struct {
	Name           string  `json:"name"`
	MaxPositionPct float64 `json:"max_position_pct"` // Max % of budget per trade
	MaxDrawdownPct float64 `json:"max_drawdown_pct"` // Max % loss before stopping
	StopLossPct    float64 `json:"stop_loss_pct"`    // Default stop-loss %
	TakeProfitPct  float64 `json:"take_profit_pct"`  // Default take-profit %
}

// PredefinedRiskProfiles contains standard risk profiles.
var PredefinedRiskProfiles = map[string]RiskProfile{
	"conservative": {
		Name:           "Conservative",
		MaxPositionPct: 0.10, // 10% max per trade
		MaxDrawdownPct: 0.10, // 10% max drawdown
		StopLossPct:    0.02, // 2% stop-loss
		TakeProfitPct:  0.04, // 4% take-profit (2:1 R:R)
	},
	"moderate": {
		Name:           "Moderate",
		MaxPositionPct: 0.20, // 20% max per trade
		MaxDrawdownPct: 0.20, // 20% max drawdown
		StopLossPct:    0.05, // 5% stop-loss
		TakeProfitPct:  0.10, // 10% take-profit
	},
	"aggressive": {
		Name:           "Aggressive",
		MaxPositionPct: 0.40, // 40% max per trade
		MaxDrawdownPct: 0.30, // 30% max drawdown
		StopLossPct:    0.10, // 10% stop-loss
		TakeProfitPct:  0.20, // 20% take-profit
	},
}

// Portfolio manages the trading account state.
type Portfolio struct {
	mu sync.RWMutex

	// Budget constraints
	InitialBudget float64 `json:"initial_budget"`
	StopLossFloor float64 `json:"stop_loss_floor"` // HARD LIMIT
	CurrentCash   float64 `json:"current_cash"`

	// Risk profile
	RiskProfile RiskProfile `json:"risk_profile"`

	// Positions and history
	Positions    map[string]*Position `json:"positions"`
	TradeHistory []Trade              `json:"trade_history"`

	// Stats
	TotalPnL  float64 `json:"total_pnl"`
	WinCount  int     `json:"win_count"`
	LossCount int     `json:"loss_count"`

	// Market reference
	market MarketDataProvider

	// Database for persistence
	db *Database
}

// NewPortfolio creates a new portfolio with budget constraints.
func NewPortfolio(budget, stopLossFloor float64, riskProfileName string, market MarketDataProvider, db *Database) *Portfolio {
	profile, ok := PredefinedRiskProfiles[riskProfileName]
	if !ok {
		profile = PredefinedRiskProfiles["conservative"]
	}

	return &Portfolio{
		InitialBudget: budget,
		StopLossFloor: stopLossFloor,
		CurrentCash:   budget,
		RiskProfile:   profile,
		Positions:     make(map[string]*Position),
		TradeHistory:  []Trade{},
		market:        market,
		db:            db,
	}
}

// GetTotalValue returns current portfolio value (cash + positions).
func (p *Portfolio) GetTotalValue() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := p.CurrentCash

	for _, pos := range p.Positions {
		price, err := p.market.GetPrice(pos.Symbol)
		if err != nil {
			continue
		}

		if pos.Side == "long" {
			total += pos.Quantity * price
		} else {
			// Short position: profit when price goes down
			pnl := (pos.EntryPrice - price) * pos.Quantity
			total += (pos.Quantity * pos.EntryPrice) + pnl
		}
	}

	return total
}

// CanTrade checks if we can open a new position of given size.
func (p *Portfolio) CanTrade(amount float64) (bool, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalValue := p.GetTotalValue()

	// Check stop-loss floor
	if totalValue <= p.StopLossFloor {
		return false, fmt.Sprintf("Portfolio at stop-loss floor ($%.2f). No trading allowed.", p.StopLossFloor)
	}

	// Check if trade would breach floor
	if totalValue-amount < p.StopLossFloor {
		maxTrade := totalValue - p.StopLossFloor
		return false, fmt.Sprintf("Trade too large. Max allowed: $%.2f to maintain floor.", maxTrade)
	}

	// Check position size limit
	maxPosition := p.InitialBudget * p.RiskProfile.MaxPositionPct
	if amount > maxPosition {
		return false, fmt.Sprintf("Trade exceeds max position size ($%.2f for %s profile).", maxPosition, p.RiskProfile.Name)
	}

	// Check available cash
	if amount > p.CurrentCash {
		return false, fmt.Sprintf("Insufficient cash. Available: $%.2f", p.CurrentCash)
	}

	return true, "OK"
}

// OpenPosition opens a new trading position.
func (p *Portfolio) OpenPosition(symbol, side string, amount float64) (*Position, error) {
	ok, reason := p.CanTrade(amount)
	if !ok {
		return nil, fmt.Errorf(reason)
	}

	price, err := p.market.GetPrice(symbol)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	quantity := amount / price
	posID := fmt.Sprintf("pos_%d", time.Now().UnixNano())

	// Calculate stop-loss and take-profit
	var stopLoss, takeProfit float64
	if side == "long" {
		stopLoss = price * (1 - p.RiskProfile.StopLossPct)
		takeProfit = price * (1 + p.RiskProfile.TakeProfitPct)
	} else {
		stopLoss = price * (1 + p.RiskProfile.StopLossPct)
		takeProfit = price * (1 - p.RiskProfile.TakeProfitPct)
	}

	pos := &Position{
		ID:         posID,
		Symbol:     symbol,
		Side:       side,
		EntryPrice: price,
		Quantity:   quantity,
		OpenTime:   time.Now(),
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
	}

	p.Positions[posID] = pos
	p.CurrentCash -= amount

	// Persist to database
	if p.db != nil {
		if err := p.db.SavePosition(pos); err != nil {
			// Log but don't fail - in-memory state is authoritative
			fmt.Printf("Warning: failed to persist position: %v\n", err)
		}
	}

	return pos, nil
}

// ClosePosition closes an existing position.
func (p *Portfolio) ClosePosition(positionID string) (*Trade, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pos, ok := p.Positions[positionID]
	if !ok {
		return nil, fmt.Errorf("position not found: %s", positionID)
	}

	price, err := p.market.GetPrice(pos.Symbol)
	if err != nil {
		return nil, err
	}

	// Calculate P&L
	var pnl float64
	if pos.Side == "long" {
		pnl = (price - pos.EntryPrice) * pos.Quantity
	} else {
		pnl = (pos.EntryPrice - price) * pos.Quantity
	}

	trade := Trade{
		ID:         pos.ID,
		Symbol:     pos.Symbol,
		Side:       pos.Side,
		EntryPrice: pos.EntryPrice,
		ExitPrice:  price,
		Quantity:   pos.Quantity,
		PnL:        pnl,
		OpenTime:   pos.OpenTime,
		CloseTime:  time.Now(),
	}

	// Update portfolio
	exitValue := pos.Quantity * price
	p.CurrentCash += exitValue
	p.TotalPnL += pnl

	if pnl > 0 {
		p.WinCount++
	} else {
		p.LossCount++
	}

	p.TradeHistory = append(p.TradeHistory, trade)
	delete(p.Positions, positionID)

	// Persist to database
	if p.db != nil {
		if err := p.db.CloseTradeInDB(&trade); err != nil {
			fmt.Printf("Warning: failed to persist trade close: %v\n", err)
		}
	}

	return &trade, nil
}

// CheckStopLoss checks if any positions have hit stop-loss and closes them.
func (p *Portfolio) CheckStopLoss() []Trade {
	p.mu.RLock()
	positionsToClose := []string{}

	for id, pos := range p.Positions {
		price, err := p.market.GetPrice(pos.Symbol)
		if err != nil {
			continue
		}

		// Check stop-loss hit
		if pos.Side == "long" && price <= pos.StopLoss {
			positionsToClose = append(positionsToClose, id)
		} else if pos.Side == "short" && price >= pos.StopLoss {
			positionsToClose = append(positionsToClose, id)
		}
	}
	p.mu.RUnlock()

	// Close positions that hit stop-loss
	closedTrades := []Trade{}
	for _, id := range positionsToClose {
		if trade, err := p.ClosePosition(id); err == nil {
			closedTrades = append(closedTrades, *trade)
		}
	}

	return closedTrades
}

// EmergencyLiquidate closes all positions (used when hitting floor).
func (p *Portfolio) EmergencyLiquidate() []Trade {
	p.mu.RLock()
	positionIDs := make([]string, 0, len(p.Positions))
	for id := range p.Positions {
		positionIDs = append(positionIDs, id)
	}
	p.mu.RUnlock()

	trades := []Trade{}
	for _, id := range positionIDs {
		if trade, err := p.ClosePosition(id); err == nil {
			trades = append(trades, *trade)
		}
	}

	return trades
}

// GetStatus returns current portfolio status.
func (p *Portfolio) GetStatus() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalValue := p.GetTotalValue()
	pnlPct := ((totalValue - p.InitialBudget) / p.InitialBudget) * 100

	positions := make([]map[string]interface{}, 0, len(p.Positions))
	for _, pos := range p.Positions {
		price, _ := p.market.GetPrice(pos.Symbol)
		var unrealizedPnL float64
		if pos.Side == "long" {
			unrealizedPnL = (price - pos.EntryPrice) * pos.Quantity
		} else {
			unrealizedPnL = (pos.EntryPrice - price) * pos.Quantity
		}

		pnlPercent := 0.0
		if pos.EntryPrice > 0 {
			pnlPercent = (unrealizedPnL / (pos.EntryPrice * pos.Quantity)) * 100
		}

		positions = append(positions, map[string]interface{}{
			"id":             pos.ID,
			"symbol":         pos.Symbol,
			"side":           pos.Side,
			"entry_price":    pos.EntryPrice,
			"current_price":  price,
			"quantity":       pos.Quantity,
			"unrealized_pnl": unrealizedPnL,
			"pnl_percent":    pnlPercent,
			"stop_loss":      pos.StopLoss,
			"take_profit":    pos.TakeProfit,
		})
	}

	winRate := 0.0
	if p.WinCount+p.LossCount > 0 {
		winRate = float64(p.WinCount) / float64(p.WinCount+p.LossCount) * 100
	}

	distanceToFloor := totalValue - p.StopLossFloor
	floorPct := (distanceToFloor / p.InitialBudget) * 100

	status := "active"
	if totalValue <= p.StopLossFloor {
		status = "STOPPED - Floor reached"
	} else if floorPct < 10 {
		status = "WARNING - Near floor"
	}

	return map[string]interface{}{
		"status":            status,
		"initial_budget":    p.InitialBudget,
		"stop_loss_floor":   p.StopLossFloor,
		"current_cash":      p.CurrentCash,
		"total_value":       totalValue,
		"total_pnl":         p.TotalPnL,
		"pnl_percent":       pnlPct,
		"distance_to_floor": distanceToFloor,
		"floor_percent":     floorPct,
		"risk_profile": map[string]interface{}{
			"name":             p.RiskProfile.Name,
			"max_position_pct": p.RiskProfile.MaxPositionPct,
			"stop_loss_pct":    p.RiskProfile.StopLossPct,
			"take_profit_pct":  p.RiskProfile.TakeProfitPct,
		},
		"open_positions": len(p.Positions),
		"positions":      positions,
		"trades_count":   len(p.TradeHistory),
		"win_count":      p.WinCount,
		"loss_count":     p.LossCount,
		"win_rate":       fmt.Sprintf("%.1f%%", winRate),
	}
}

// ============================================================================
// PORTFOLIO TOOLS
// ============================================================================

// CreateGetPortfolioStatusTool creates a tool to get portfolio status.
func CreateGetPortfolioStatusTool(portfolio *Portfolio) core.Tool {
	return tools.New("get_trading_status").
		Description("Get current trading portfolio status including balance, positions, P&L, and risk metrics").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			return portfolio.GetStatus(), nil
		}).
		Build()
}

// CreateOpenPositionTool creates a tool to open a position (requires confirmation).
func CreateOpenPositionTool(portfolio *Portfolio) core.Tool {
	return tools.New("open_trade").
		Description("Open a new trading position. Requires user confirmation.").
		RequiresConfirmation().
		SummaryTemplate("Authorize New Position").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol": tools.StringProperty("Trading symbol (e.g., BTCUSDT)"),
			"side":   tools.StringProperty("Trade direction: 'long' (buy) or 'short' (sell)"),
			"amount": tools.NumberProperty("Dollar amount to trade"),
		}, "symbol", "side", "amount")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol string  `json:"symbol"`
				Side   string  `json:"side"`
				Amount float64 `json:"amount"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			pos, err := portfolio.OpenPosition(params.Symbol, params.Side, params.Amount)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"success":     true,
				"position_id": pos.ID,
				"symbol":      pos.Symbol,
				"side":        pos.Side,
				"entry_price": fmt.Sprintf("%.4f", pos.EntryPrice),
				"quantity":    fmt.Sprintf("%.6f", pos.Quantity),
				"stop_loss":   fmt.Sprintf("%.4f", pos.StopLoss),
				"take_profit": fmt.Sprintf("%.4f", pos.TakeProfit),
			}, nil
		}).
		Build()
}

// CreateClosePositionTool creates a tool to close a position (requires confirmation).
func CreateClosePositionTool(portfolio *Portfolio) core.Tool {
	return tools.New("close_trade").
		Description("Close an existing trading position. Requires user confirmation.").
		RequiresConfirmation().
		SummaryTemplate("Authorize Closing Selected Position").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"position_id": tools.StringProperty("ID of the position to close"),
			"symbol":      tools.StringProperty("Symbol of the position (e.g., BTCUSDT) for confirmation"),
		}, "position_id", "symbol")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				PositionID string `json:"position_id"`
				Symbol     string `json:"symbol"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			trade, err := portfolio.ClosePosition(params.PositionID)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"success":     true,
				"symbol":      trade.Symbol,
				"side":        trade.Side,
				"entry_price": fmt.Sprintf("%.4f", trade.EntryPrice),
				"exit_price":  fmt.Sprintf("%.4f", trade.ExitPrice),
				"pnl":         fmt.Sprintf("%.2f", trade.PnL),
			}, nil
		}).
		Build()
}
