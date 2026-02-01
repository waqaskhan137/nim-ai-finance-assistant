package trading

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ============================================================================
// AUTONOMOUS TRADING ENGINE
// ============================================================================

// AnalysisResult holds the result of market analysis for an asset.
type AnalysisResult struct {
	Symbol      string          `json:"symbol"`
	Timestamp   time.Time       `json:"timestamp"`
	Price       float64         `json:"price"`
	RSI         float64         `json:"rsi"`
	SMA20       float64         `json:"sma20"`
	MACD        float64         `json:"macd"`
	MACDSignal  float64         `json:"macd_signal"`
	MACDHist    float64         `json:"macd_histogram"`
	Bollinger   *BollingerBands `json:"bollinger"`
	TrendSignal string          `json:"trend_signal"` // "bullish", "bearish", "neutral"
	Strength    string          `json:"strength"`     // "strong", "moderate", "weak"
}

// TradeSignal represents a trading recommendation.
type TradeSignal struct {
	Symbol       string  `json:"symbol"`
	Action       string  `json:"action"`     // "buy", "sell", "hold"
	Side         string  `json:"side"`       // "long", "short"
	Confidence   string  `json:"confidence"` // "high", "medium", "low"
	EntryPrice   float64 `json:"entry_price"`
	StopLoss     float64 `json:"stop_loss"`
	TakeProfit   float64 `json:"take_profit"`
	PositionSize float64 `json:"position_size"` // As % of available funds
	Reasoning    string  `json:"reasoning"`
}

// TradingDecision records a decision made by the auto-trader.
type TradingDecision struct {
	Timestamp   time.Time       `json:"timestamp"`
	Symbol      string          `json:"symbol"`
	Action      string          `json:"action"`
	Reasoning   string          `json:"reasoning"`
	Analysis    *AnalysisResult `json:"analysis,omitempty"`
	Signal      *TradeSignal    `json:"signal,omitempty"`
	Executed    bool            `json:"executed"`
	ErrorReason string          `json:"error_reason,omitempty"`
}

// AutoTraderStatus represents the current state of the auto-trader.
type AutoTraderStatus struct {
	Running         bool                       `json:"running"`
	StartedAt       *time.Time                 `json:"started_at,omitempty"`
	Interval        string                     `json:"interval"`
	LoopCount       int                        `json:"loop_count"`
	AssetsMonitored []string                   `json:"assets_monitored"`
	LastLoopAt      *time.Time                 `json:"last_loop_at,omitempty"`
	CurrentAnalysis map[string]*AnalysisResult `json:"current_analysis,omitempty"`
	RecentDecisions []TradingDecision          `json:"recent_decisions,omitempty"`
	OpenPositions   int                        `json:"open_positions"`
	TotalPnL        float64                    `json:"total_pnl"`
	Message         string                     `json:"message"`
}

// AutoTrader manages autonomous trading based on user preferences.
type AutoTrader struct {
	TradingSystem *TradingSystem
	Preferences   *TradingPreferences
	Running       bool
	Interval      time.Duration
	stopChan      chan struct{}
	mu            sync.RWMutex

	// State
	startedAt      *time.Time
	loopCount      int
	lastLoopAt     *time.Time
	LastAnalysis   map[string]*AnalysisResult
	DecisionLog    []TradingDecision
	maxDecisionLog int

	// Notify is a callback for real-time updates
	Notify func(event string, data interface{})
}

// NewAutoTrader creates a new autonomous trading engine.
func NewAutoTrader(ts *TradingSystem, prefs *TradingPreferences) *AutoTrader {
	interval := 1 * time.Minute // Default: analyze every minute
	if prefs.Style == "hft" {
		interval = 10 * time.Second // High-frequency: every 10 seconds
	} else if prefs.Style == "day_trading" {
		interval = 30 * time.Second // More frequent for day trading
	} else if prefs.Style == "hold" {
		interval = 5 * time.Minute // Less frequent for holding
	}

	return &AutoTrader{
		TradingSystem:  ts,
		Preferences:    prefs,
		Interval:       interval,
		LastAnalysis:   make(map[string]*AnalysisResult),
		DecisionLog:    make([]TradingDecision, 0),
		maxDecisionLog: 50, // Keep last 50 decisions
	}
}

// Start begins autonomous trading.
func (at *AutoTrader) Start(ctx context.Context) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	if at.Running {
		return fmt.Errorf("auto-trading is already running")
	}

	// Validate we have preferences
	if at.Preferences == nil || len(at.Preferences.Assets) == 0 {
		return fmt.Errorf("trading preferences not set - use set_trading_preferences first")
	}

	// Validate we can trade
	if !at.Preferences.AutoTrade {
		return fmt.Errorf("auto-trade is disabled in preferences - enable it first")
	}

	at.Running = true
	now := time.Now()
	at.startedAt = &now
	at.stopChan = make(chan struct{})
	at.loopCount = 0

	// Start the trading loop in a goroutine
	// Use background context so it doesn't die when tool call returns
	go at.tradingLoop(context.Background())

	log.Printf("🤖 Auto-trading started! Monitoring %v every %v", at.Preferences.Assets, at.Interval)
	return nil
}

// Stop halts autonomous trading.
func (at *AutoTrader) Stop() error {
	at.mu.Lock()
	defer at.mu.Unlock()

	if !at.Running {
		return fmt.Errorf("auto-trading is not running")
	}

	close(at.stopChan)
	at.Running = false

	log.Printf("🛑 Auto-trading stopped after %d loops", at.loopCount)
	return nil
}

// tradingLoop is the main autonomous trading loop.
func (at *AutoTrader) tradingLoop(ctx context.Context) {
	ticker := time.NewTicker(at.Interval)
	defer ticker.Stop()

	// Run immediately on start
	at.executeLoop(ctx)

	for {
		select {
		case <-ctx.Done():
			at.mu.Lock()
			at.Running = false
			at.mu.Unlock()
			return
		case <-at.stopChan:
			return
		case <-ticker.C:
			at.executeLoop(ctx)
		}
	}
}

// executeLoop runs one iteration of the trading loop.
func (at *AutoTrader) executeLoop(ctx context.Context) {
	at.mu.Lock()
	at.loopCount++
	now := time.Now()
	at.lastLoopAt = &now
	prefs := at.Preferences
	at.mu.Unlock()

	log.Printf("🔄 Auto-trading loop #%d starting...", at.loopCount)

	// Check if we should still be trading
	if !at.shouldContinueTrading() {
		log.Printf("⚠️ Trading conditions not met, skipping loop")
		return
	}

	// Analyze each asset
	for _, symbol := range prefs.Assets {
		// Analyze the asset
		analysis, err := at.analyzeAsset(symbol)
		if err != nil {
			log.Printf("❌ Failed to analyze %s: %v", symbol, err)
			continue
		}

		at.mu.Lock()
		at.LastAnalysis[symbol] = analysis
		at.mu.Unlock()

		// Evaluate for trade signals
		signal := at.evaluateTrade(symbol, analysis)

		// Log the decision
		decision := TradingDecision{
			Timestamp: time.Now(),
			Symbol:    symbol,
			Action:    signal.Action,
			Reasoning: signal.Reasoning,
			Analysis:  analysis,
			Signal:    signal,
			Executed:  false,
		}

		// Execute trade if conditions met
		if signal.Action != "hold" && signal.Confidence != "low" {
			err := at.executeTrade(signal)
			if err != nil {
				decision.ErrorReason = err.Error()
				log.Printf("❌ Trade execution failed for %s: %v", symbol, err)
			} else {
				decision.Executed = true
				log.Printf("✅ Trade executed for %s: %s %s", symbol, signal.Action, signal.Side)
			}
		}

		at.addDecision(decision)
	}

	// Check exit conditions for existing positions
	at.checkExitConditions()

	// Check profit targets
	at.checkProfitTargets()

	log.Printf("✅ Auto-trading loop #%d complete", at.loopCount)
}

// checkProfitTargets checks if total P&L has hit user's target.
func (at *AutoTrader) checkProfitTargets() {
	status := at.TradingSystem.Portfolio.GetStatus()
	totalPnL, ok := status["total_pnl"].(float64)
	if !ok {
		return
	}
	initialBudget, ok := status["initial_budget"].(float64)
	if !ok || initialBudget == 0 {
		return
	}

	profitPct := totalPnL / initialBudget

	if profitPct >= at.Preferences.ProfitTarget {
		log.Printf("🎉 PROFIT TARGET REACHED! Current P&L: $%.2f (%.1f%%) >= Target: %.1f%%",
			totalPnL, profitPct*100, at.Preferences.ProfitTarget*100)
		log.Printf("💡 Recommendation: Use 'withdraw trading profits' to secure your gains.")

		if at.Notify != nil {
			at.Notify("profit_target_reached", map[string]interface{}{
				"current_pnl":  totalPnL,
				"profit_pct":   profitPct * 100,
				"target_pct":   at.Preferences.ProfitTarget * 100,
				"current_cash": status["current_cash"],
			})
		}
	}
}

// shouldContinueTrading checks if trading should continue.
func (at *AutoTrader) shouldContinueTrading() bool {
	// Check portfolio health
	status := at.TradingSystem.Portfolio.GetStatus()

	canTrade, ok := status["can_trade"].(bool)
	if ok && !canTrade {
		log.Printf("⚠️ Cannot trade - portfolio at floor limit")
		return false
	}

	return true
}

// analyzeAsset performs technical analysis on a symbol.
func (at *AutoTrader) analyzeAsset(symbol string) (*AnalysisResult, error) {
	market := at.TradingSystem.Market

	// Get current price
	price, err := market.GetPrice(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}

	// Get candles for indicators
	candles, err := market.GetCandles(symbol, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get candles: %w", err)
	}

	// Extract close prices
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate indicators
	rsi := CalculateRSI(closes, 14)
	sma20 := CalculateSMA(closes, 20)
	macd, signal, histogram := CalculateMACD(closes)
	bb := CalculateBollingerBands(closes, 20, 2)

	// Determine trend
	trendSignal := "neutral"
	strength := "weak"

	bullishCount := 0
	bearishCount := 0

	// RSI analysis
	if rsi < 30 {
		bullishCount += 2 // Oversold = bullish
	} else if rsi < 40 {
		bullishCount += 1
	} else if rsi > 70 {
		bearishCount += 2 // Overbought = bearish
	} else if rsi > 60 {
		bearishCount += 1
	}

	// Trend analysis (price vs SMA)
	if price > sma20 {
		bullishCount += 1
	} else {
		bearishCount += 1
	}

	// MACD analysis
	if histogram > 0 {
		bullishCount += 1
	} else {
		bearishCount += 1
	}

	// Determine overall trend
	if bullishCount > bearishCount+1 {
		trendSignal = "bullish"
	} else if bearishCount > bullishCount+1 {
		trendSignal = "bearish"
	}

	// Determine strength
	maxCount := bullishCount
	if bearishCount > bullishCount {
		maxCount = bearishCount
	}
	if maxCount >= 4 {
		strength = "strong"
	} else if maxCount >= 2 {
		strength = "moderate"
	}

	return &AnalysisResult{
		Symbol:      symbol,
		Timestamp:   time.Now(),
		Price:       price,
		RSI:         rsi,
		SMA20:       sma20,
		MACD:        macd,
		MACDSignal:  signal,
		MACDHist:    histogram,
		Bollinger:   &bb,
		TrendSignal: trendSignal,
		Strength:    strength,
	}, nil
}

// evaluateTrade generates a trade signal based on analysis.
func (at *AutoTrader) evaluateTrade(symbol string, analysis *AnalysisResult) *TradeSignal {
	prefs := at.Preferences
	riskProfile := PredefinedRiskProfiles[prefs.RiskProfile]

	signal := &TradeSignal{
		Symbol:     symbol,
		Action:     "hold",
		EntryPrice: analysis.Price,
	}

	var reasons []string

	// Determine thresholds based on style
	buyThreshold := 35.0
	sellThreshold := 65.0

	if prefs.Style == "hft" || prefs.RiskProfile == "aggressive" {
		// Looser thresholds for demo/HFT
		buyThreshold = 48.0
		sellThreshold = 52.0
	} else if prefs.Style == "day_trading" {
		buyThreshold = 40.0
		sellThreshold = 60.0
	}

	// Check for long entry
	if analysis.RSI < buyThreshold {
		if analysis.RSI < buyThreshold-10 {
			// Extreme oversold - high confidence entry
			signal.Action = "buy"
			signal.Side = "long"
			signal.Confidence = "high"
			reasons = append(reasons, fmt.Sprintf("RSI extremely oversold at %.1f", analysis.RSI))
		} else if analysis.Price > analysis.SMA20 || analysis.MACDHist > 0 {
			// Oversold with confirmation
			signal.Action = "buy"
			signal.Side = "long"
			signal.Confidence = "medium"
			reasons = append(reasons, fmt.Sprintf("RSI oversold at %.1f with trend confirmation", analysis.RSI))
		} else if prefs.Style == "hft" {
			// HFT takes risk even without confirmation
			signal.Action = "buy"
			signal.Side = "long"
			signal.Confidence = "medium"
			reasons = append(reasons, fmt.Sprintf("HFT Mode: Buy on RSI %.1f dip", analysis.RSI))
		} else {
			signal.Confidence = "low"
			reasons = append(reasons, fmt.Sprintf("RSI oversold at %.1f but no confirmation", analysis.RSI))
		}
	}

	// Check for short entry (if not already going long)
	if signal.Action == "hold" && analysis.RSI > sellThreshold {
		if analysis.RSI > sellThreshold+10 {
			// Extreme overbought - high confidence entry
			signal.Action = "sell"
			signal.Side = "short"
			signal.Confidence = "high"
			reasons = append(reasons, fmt.Sprintf("RSI extremely overbought at %.1f", analysis.RSI))
		} else if analysis.Price < analysis.SMA20 || analysis.MACDHist < 0 {
			// Overbought with confirmation
			signal.Action = "sell"
			signal.Side = "short"
			signal.Confidence = "medium"
			reasons = append(reasons, fmt.Sprintf("RSI overbought at %.1f with trend confirmation", analysis.RSI))
		} else if prefs.Style == "hft" {
			// HFT takes risk even without confirmation
			signal.Action = "sell"
			signal.Side = "short"
			signal.Confidence = "medium"
			reasons = append(reasons, fmt.Sprintf("HFT Mode: Sell on RSI %.1f spike", analysis.RSI))
		} else {
			signal.Confidence = "low"
			reasons = append(reasons, fmt.Sprintf("RSI overbought at %.1f but no confirmation", analysis.RSI))
		}
	}

	// If still holding, explain why
	if signal.Action == "hold" {
		reasons = append(reasons, fmt.Sprintf("RSI at %.1f is neutral, waiting for better entry", analysis.RSI))
		signal.Confidence = "none"
	}

	// Calculate stop-loss and take-profit
	if signal.Action != "hold" {
		if signal.Side == "long" {
			signal.StopLoss = analysis.Price * (1 - riskProfile.StopLossPct)
			signal.TakeProfit = analysis.Price * (1 + riskProfile.TakeProfitPct)
		} else {
			signal.StopLoss = analysis.Price * (1 + riskProfile.StopLossPct)
			signal.TakeProfit = analysis.Price * (1 - riskProfile.TakeProfitPct)
		}
		signal.PositionSize = riskProfile.MaxPositionPct
	}

	signal.Reasoning = fmt.Sprintf("%s | Trend: %s (%s)",
		joinReasons(reasons), analysis.TrendSignal, analysis.Strength)

	return signal
}

// executeTrade executes a trade signal.
func (at *AutoTrader) executeTrade(signal *TradeSignal) error {
	portfolio := at.TradingSystem.Portfolio

	// Calculate position size in dollars
	status := portfolio.GetStatus()
	availableCash, ok := status["available_cash"].(float64)
	if !ok {
		return fmt.Errorf("could not get available cash")
	}

	positionAmount := availableCash * signal.PositionSize

	// Check if we can trade this amount
	canTrade, reason := portfolio.CanTrade(positionAmount)
	if !canTrade {
		return fmt.Errorf("risk check failed: %s", reason)
	}

	// Execute the trade
	_, err := portfolio.OpenPosition(signal.Symbol, signal.Side, positionAmount)
	if err != nil {
		return fmt.Errorf("failed to open position: %w", err)
	}

	log.Printf("📈 Opened %s %s position: $%.2f at $%.2f (SL: $%.2f, TP: $%.2f)",
		signal.Side, signal.Symbol, positionAmount, signal.EntryPrice,
		signal.StopLoss, signal.TakeProfit)

	if at.Notify != nil {
		at.Notify("trade_opened", map[string]interface{}{
			"symbol":      signal.Symbol,
			"side":        signal.Side,
			"amount":      positionAmount,
			"entry_price": signal.EntryPrice,
			"stop_loss":   signal.StopLoss,
			"take_profit": signal.TakeProfit,
		})
	}

	return nil
}

// checkExitConditions monitors positions for stop-loss/take-profit.
func (at *AutoTrader) checkExitConditions() {
	closedTrades := at.TradingSystem.Portfolio.CheckStopLoss()

	for _, trade := range closedTrades {
		log.Printf("🔔 Position closed: %s %s - P&L: $%.2f",
			trade.Symbol, trade.Side, trade.PnL)

		decision := TradingDecision{
			Timestamp: time.Now(),
			Symbol:    trade.Symbol,
			Action:    "close",
			Reasoning: fmt.Sprintf("Exit triggered - P&L: $%.2f", trade.PnL),
			Executed:  true,
		}
		at.addDecision(decision)

		if at.Notify != nil {
			at.Notify("trade_closed", map[string]interface{}{
				"symbol":     trade.Symbol,
				"side":       trade.Side,
				"pnl":        trade.PnL,
				"exit_price": trade.ExitPrice,
				"reason":     "Stop-loss/Take-profit triggered",
			})
		}
	}
}

// addDecision adds a decision to the log, maintaining max size.
func (at *AutoTrader) addDecision(decision TradingDecision) {
	at.mu.Lock()
	defer at.mu.Unlock()

	at.DecisionLog = append(at.DecisionLog, decision)

	// Trim if over max
	if len(at.DecisionLog) > at.maxDecisionLog {
		at.DecisionLog = at.DecisionLog[len(at.DecisionLog)-at.maxDecisionLog:]
	}
}

// GetStatus returns the current auto-trader status.
func (at *AutoTrader) GetStatus() *AutoTraderStatus {
	at.mu.RLock()
	defer at.mu.RUnlock()

	status := &AutoTraderStatus{
		Running:         at.Running,
		StartedAt:       at.startedAt,
		Interval:        at.Interval.String(),
		LoopCount:       at.loopCount,
		LastLoopAt:      at.lastLoopAt,
		CurrentAnalysis: at.LastAnalysis,
		OpenPositions:   len(at.TradingSystem.Portfolio.Positions),
	}

	if at.Preferences != nil {
		status.AssetsMonitored = at.Preferences.Assets
	}

	// Get recent decisions (last 10)
	if len(at.DecisionLog) > 0 {
		start := len(at.DecisionLog) - 10
		if start < 0 {
			start = 0
		}
		status.RecentDecisions = at.DecisionLog[start:]
	}

	// Get total P&L
	portfolioStatus := at.TradingSystem.Portfolio.GetStatus()
	if pnl, ok := portfolioStatus["total_pnl"].(float64); ok {
		status.TotalPnL = pnl
	}

	if at.Running {
		status.Message = fmt.Sprintf("Auto-trading running. Loop #%d, monitoring %d assets.",
			at.loopCount, len(status.AssetsMonitored))
	} else {
		status.Message = "Auto-trading is stopped."
	}

	return status
}

// UpdatePreferences updates the trading preferences.
func (at *AutoTrader) UpdatePreferences(prefs *TradingPreferences) {
	at.mu.Lock()
	defer at.mu.Unlock()
	at.Preferences = prefs

	// Update interval based on style
	if prefs.Style == "hft" {
		at.Interval = 10 * time.Second // High-frequency: every 10 seconds
	} else if prefs.Style == "day_trading" {
		at.Interval = 30 * time.Second
	} else if prefs.Style == "hold" {
		at.Interval = 5 * time.Minute
	} else {
		at.Interval = 1 * time.Minute
	}
}

// Helper function to join reasons
func joinReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "No specific reason"
	}
	result := reasons[0]
	for i := 1; i < len(reasons); i++ {
		result += "; " + reasons[i]
	}
	return result
}
