package trading

import (
	"github.com/becomeliminal/nim-go-sdk/engine"
	"github.com/becomeliminal/nim-go-sdk/subagent"
)

// ============================================================================
// MARKET ANALYST SUBAGENT
// ============================================================================

// MarketAnalystSystemPrompt defines the analyst's behavior.
const MarketAnalystSystemPrompt = `You are a professional market analyst AI specializing in technical analysis.

YOUR ROLE:
- Analyze price data and technical indicators
- Identify trading opportunities (entries/exits)
- Assess trend direction and strength
- Provide clear, actionable signals

ANALYSIS APPROACH:
1. Check current price and recent trend
2. Calculate RSI for overbought/oversold conditions
3. Check MACD for momentum confirmation
4. Consider Bollinger Bands for volatility

SIGNAL FORMAT:
When providing analysis, always include:
- Current price and trend direction
- RSI reading and interpretation
- MACD momentum assessment
- Overall recommendation: STRONG BUY, BUY, HOLD, SELL, STRONG SELL
- Confidence level: HIGH, MEDIUM, LOW

RULES:
- Be objective and data-driven
- Never make trades - you only analyze
- Always explain your reasoning
- Warn about high volatility or uncertainty

Available tools: get_market_price, get_candles, calc_indicators`

// NewMarketAnalyst creates a market analysis subagent.
func NewMarketAnalyst(eng *engine.Engine) *subagent.SubAgent {
	return subagent.NewSubAgent(eng, subagent.SubAgentConfig{
		Name:         "market_analyst",
		SystemPrompt: MarketAnalystSystemPrompt,
		AvailableTools: []string{
			"get_market_price",
			"get_candles",
			"calc_indicators",
		},
		MaxTurns:  5,
		MaxTokens: 1024,
	})
}

// NewMarketAnalystDelegationTool creates a delegation tool for market analysis.
func NewMarketAnalystDelegationTool(eng *engine.Engine) *subagent.DelegationTool {
	return subagent.NewDelegationTool(subagent.DelegationConfig{
		SubAgent:    NewMarketAnalyst(eng),
		ToolName:    "analyze_market",
		Description: "Delegate market analysis to the analyst specialist. Use for technical analysis, trend identification, and trade signals.",
		TaskFormatter: func(query string) string {
			return "Perform technical analysis and provide trading signals for: " + query
		},
		QueryDescription: "The market/symbol to analyze (e.g., 'BTCUSDT', 'analyze Bitcoin for entry', 'is ETH oversold?')",
	})
}

// ============================================================================
// RISK MANAGER SUBAGENT
// ============================================================================

// RiskManagerSystemPrompt defines the risk manager's behavior.
const RiskManagerSystemPrompt = `You are a risk management AI responsible for protecting the trading portfolio.

YOUR ROLE:
- Monitor portfolio health and exposure
- Enforce budget constraints and stop-loss rules
- Calculate appropriate position sizes
- Warn about dangerous trading conditions

RISK ASSESSMENT:
1. Check current portfolio value vs floor limit
2. Calculate available trading capacity
3. Assess current position exposure
4. Evaluate risk/reward of proposed trades

POSITION SIZING RULES:
- Never risk more than the max position % for the risk profile
- Always maintain buffer above stop-loss floor
- Consider existing position exposure

ALERTS - Issue warnings for:
- Portfolio approaching stop-loss floor (< 20% buffer)
- High drawdown from initial budget
- Concentrated positions (too much in one asset)
- Correlated risk exposure

OUTPUT FORMAT:
- Current portfolio status
- Risk level: LOW, MEDIUM, HIGH, CRITICAL
- Available for trading: $X
- Recommended max position size
- Any warnings or concerns

RULES:
- Be conservative - capital preservation is priority
- Never approve trades that could breach floor
- Recommend closing positions if risk is too high
- Always quantify the risk in dollar terms

Available tools: get_trading_status`

// NewRiskManager creates a risk management subagent.
func NewRiskManager(eng *engine.Engine) *subagent.SubAgent {
	return subagent.NewSubAgent(eng, subagent.SubAgentConfig{
		Name:         "risk_manager",
		SystemPrompt: RiskManagerSystemPrompt,
		AvailableTools: []string{
			"get_trading_status",
		},
		MaxTurns:  3, // Should be quick decisions
		MaxTokens: 768,
	})
}

// NewRiskManagerDelegationTool creates a delegation tool for risk assessment.
func NewRiskManagerDelegationTool(eng *engine.Engine) *subagent.DelegationTool {
	return subagent.NewDelegationTool(subagent.DelegationConfig{
		SubAgent:    NewRiskManager(eng),
		ToolName:    "assess_risk",
		Description: "Delegate risk assessment to the risk manager. Use before trades to check position sizing and portfolio health.",
		TaskFormatter: func(query string) string {
			return "Assess risk and provide guidance on: " + query
		},
		QueryDescription: "The risk question (e.g., 'Can I open a $2 position?', 'What's my current risk level?', 'Should I close any positions?')",
	})
}

// ============================================================================
// STRATEGY EVALUATOR SUBAGENT
// ============================================================================

// StrategyEvaluatorSystemPrompt defines the strategy evaluator's behavior.
const StrategyEvaluatorSystemPrompt = `You are a trading strategy evaluator AI that decides when to execute trades.

YOUR ROLE:
- Evaluate trade signals from the market analyst
- Match signals with the user's risk profile
- Determine optimal entry/exit points
- Provide trade recommendations

STRATEGY: RSI Reversal with Trend Confirmation
Entry Rules for LONG:
- RSI below 35 (oversold)
- Price above 20-period SMA (uptrend) OR RSI below 25 (extreme oversold)
- MACD showing bullish momentum or about to cross

Entry Rules for SHORT:
- RSI above 65 (overbought)
- Price below 20-period SMA (downtrend) OR RSI above 75 (extreme overbought)
- MACD showing bearish momentum

EXIT RULES:
- Take profit: 4% for conservative, 10% for moderate, 20% for aggressive
- Stop loss: 2% for conservative, 5% for moderate, 10% for aggressive
- Also exit when RSI reaches opposite extreme

DECISION FORMAT:
- Trade signal: BUY, SELL, or WAIT
- Entry price recommendation
- Stop-loss price
- Take-profit price
- Position size recommendation (% of available funds)
- Confidence: HIGH, MEDIUM, LOW
- Reasoning

RULES:
- Only recommend trades when multiple indicators align
- WAIT is often the best decision
- Quality over quantity
- Never recommend trades you wouldn't take yourself

Available tools: calc_indicators, get_trading_status`

// NewStrategyEvaluator creates a strategy evaluation subagent.
func NewStrategyEvaluator(eng *engine.Engine) *subagent.SubAgent {
	return subagent.NewSubAgent(eng, subagent.SubAgentConfig{
		Name:         "strategy_evaluator",
		SystemPrompt: StrategyEvaluatorSystemPrompt,
		AvailableTools: []string{
			"calc_indicators",
			"get_trading_status",
		},
		MaxTurns:  4,
		MaxTokens: 1024,
	})
}

// NewStrategyEvaluatorDelegationTool creates a delegation tool for strategy evaluation.
func NewStrategyEvaluatorDelegationTool(eng *engine.Engine) *subagent.DelegationTool {
	return subagent.NewDelegationTool(subagent.DelegationConfig{
		SubAgent:    NewStrategyEvaluator(eng),
		ToolName:    "evaluate_trade",
		Description: "Delegate trade evaluation to the strategy specialist. Use to get specific entry/exit recommendations.",
		TaskFormatter: func(query string) string {
			return "Evaluate this trading opportunity and provide specific recommendations: " + query
		},
		QueryDescription: "The trade to evaluate (e.g., 'Should I buy BTCUSDT now?', 'Evaluate ETH long entry')",
	})
}
