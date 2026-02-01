package trading

import (
	"log"
	"os"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/engine"
	"github.com/becomeliminal/nim-go-sdk/subagent"

	"github.com/becomeliminal/nim-go-sdk/examples/hackathon-starter/trading/connectors"
)

// ============================================================================
// TRADING ORCHESTRATOR
// ============================================================================

// TradingSystem holds all components of the trading system.
type TradingSystem struct {
	// Market data
	Market MarketDataProvider

	// Portfolio management
	Portfolio *Portfolio

	// Database for persistence
	DB *Database

	// Exchange connector (Binance, etc.)
	Binance *connectors.BinanceConnector

	// Autonomous trading engine
	AutoTrader *AutoTrader

	// Subagents (created when engine is available)
	AnalystTool  *subagent.DelegationTool
	RiskTool     *subagent.DelegationTool
	StrategyTool *subagent.DelegationTool
}

// NewTradingSystem creates a complete trading system.
func NewTradingSystem(budget, stopLossFloor float64, riskProfile string) *TradingSystem {
	// Initialize Binance connector if credentials are available
	var binance *connectors.BinanceConnector
	apiKey := os.Getenv("BINANCE_API_KEY")
	apiSecret := os.Getenv("BINANCE_API_SECRET")
	if apiKey != "" && apiSecret != "" {
		testnet := os.Getenv("BINANCE_TESTNET") == "true"
		binance = connectors.NewBinanceConnector(connectors.BinanceConfig{
			APIKey:    apiKey,
			APISecret: apiSecret,
			Testnet:   testnet,
		})
		if testnet {
			log.Println("✅ Binance Testnet connector initialized (paper trading)")
		} else {
			log.Println("✅ Binance connector initialized (LIVE TRADING)")
		}
	}

	// Initialize market data provider - use real Binance data if available
	var market MarketDataProvider
	if binance != nil {
		market = NewRealMarketData(binance)
		log.Println("✅ Using real Binance market data")
	} else {
		market = NewMockMarketData()
		log.Println("ℹ️ Using mock market data (set BINANCE_API_KEY and BINANCE_API_SECRET for real data)")
	}

	// Initialize database
	db, err := NewDatabase("trading.db")
	if err != nil {
		log.Printf("⚠️ Database initialization failed: %v (trades will not be persisted)", err)
	} else {
		log.Println("✅ Trading database initialized (trading.db)")
	}

	portfolio := NewPortfolio(budget, stopLossFloor, riskProfile, market, db)

	return &TradingSystem{
		Market:    market,
		Portfolio: portfolio,
		DB:        db,
		Binance:   binance,
	}
}

// GetBinanceTools returns Binance-specific tools if connector is available.
func (ts *TradingSystem) GetBinanceTools() []core.Tool {
	if ts.Binance == nil {
		return nil
	}
	return connectors.CreateBinanceTools(ts.Binance)
}

// InitializeSubagents sets up the subagent delegation tools.
// This must be called after the engine is created.
func (ts *TradingSystem) InitializeSubagents(eng *engine.Engine) {
	ts.AnalystTool = NewMarketAnalystDelegationTool(eng)
	ts.RiskTool = NewRiskManagerDelegationTool(eng)
	ts.StrategyTool = NewStrategyEvaluatorDelegationTool(eng)
}

// GetAllTools returns all trading tools for registration with the server.
func (ts *TradingSystem) GetAllTools() []core.Tool {
	tools := []core.Tool{
		// Market data tools
		CreateGetPriceTool(ts.Market),
		CreateGetCandlesTool(ts.Market),
		CreateCalcIndicatorsTool(ts.Market),

		// Portfolio tools
		CreateGetPortfolioStatusTool(ts.Portfolio),
		CreateOpenPositionTool(ts.Portfolio),
		CreateClosePositionTool(ts.Portfolio),
		CreateCloseAllTradesTool(ts.Portfolio),
	}

	return tools
}

// GetSubagentTools returns the subagent delegation tools.
// Call this after InitializeSubagents.
func (ts *TradingSystem) GetSubagentTools() []core.Tool {
	if ts.AnalystTool == nil {
		return nil
	}

	return []core.Tool{
		ts.AnalystTool,
		ts.RiskTool,
		ts.StrategyTool,
	}
}

// ============================================================================
// TRADING SYSTEM PROMPT ADDITIONS
// ============================================================================

// TradingSystemPrompt is the system prompt addition for trading capabilities.
const TradingSystemPrompt = `
TRADING CAPABILITIES:
You now have autonomous trading capabilities. You can analyze markets, manage a portfolio, and execute trades with proper risk management.

TRADING SUBAGENTS (Specialists you can delegate to):
1. analyze_market - Technical analysis specialist
   - Use for: Market trends, RSI/MACD analysis, entry/exit signals
   - Example: "analyze_market: Is BTCUSDT a good buy right now?"

2. assess_risk - Risk management specialist  
   - Use for: Portfolio health, position sizing, stop-loss checks
   - Example: "assess_risk: Can I open a $2 position safely?"

3. evaluate_trade - Strategy specialist
   - Use for: Specific trade recommendations, entry/exit prices
   - Example: "evaluate_trade: Should I long ETHUSDT?"

DIRECT TRADING TOOLS:
- get_market_price: Get current price for a symbol
- get_candles: Get historical OHLCV data
- calc_indicators: Calculate RSI, MACD, Bollinger Bands
- get_trading_status: Check portfolio, positions, P&L
- open_trade: Open a new position (REQUIRES CONFIRMATION)
- close_trade: Close an existing position (REQUIRES CONFIRMATION)

AUTONOMOUS TRADING TOOLS:
- start_auto_trading: Start autonomous trading based on your preferences
- stop_auto_trading: Stop autonomous trading (optionally close positions)
- get_auto_trading_status: Check auto-trading status and recent decisions

TRADING WORKFLOW:
1. When user sets a budget, initialize the trading system
2. Set trading preferences (assets, style, risk profile)
3. For manual trading: consult analysts, then execute trades with confirmation
4. For auto-trading: start_auto_trading to let the system trade automatically
5. Monitor with get_auto_trading_status

RISK RULES (CRITICAL):
- NEVER trade below the stop-loss floor
- ALWAYS confirm manual trades with the user
- Auto-trading respects risk profile limits
- Close positions if hitting stop-loss levels

AVAILABLE SYMBOLS:
- BTCUSDT (Bitcoin/USDT)
- ETHUSDT (Ethereum/USDT)  
- XAUUSD (Gold/USD)
- EURUSD (Euro/USD forex)
`

// DefaultTradingConfig provides default trading parameters.
type DefaultTradingConfig struct {
	DefaultBudget        float64
	DefaultStopLossFloor float64
	DefaultRiskProfile   string
}

// GetDefaultConfig returns sensible defaults.
func GetDefaultConfig() DefaultTradingConfig {
	return DefaultTradingConfig{
		DefaultBudget:        10.0,
		DefaultStopLossFloor: 7.0,
		DefaultRiskProfile:   "conservative",
	}
}
