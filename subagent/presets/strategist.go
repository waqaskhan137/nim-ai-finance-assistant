// Package presets provides pre-configured sub-agents for common tasks.
package presets

import (
	"github.com/becomeliminal/nim-go-sdk/engine"
	"github.com/becomeliminal/nim-go-sdk/subagent"
)

// StrategistSystemPrompt is the system prompt for the market strategist sub-agent.
// This agent implements the evaluator-optimizer pattern from Anthropic's
// "Building Effective Agents" guidance - it analyzes market data and provides
// actionable investment strategies with iterative refinement.
const StrategistSystemPrompt = `You are a market strategist specialist.

Your role is to analyze market conditions and develop investment strategies.
Focus on:
- Market trend analysis and momentum indicators
- Risk assessment and position sizing recommendations
- Entry/exit timing based on technical signals
- Portfolio allocation suggestions
- Correlation analysis between assets

METHODOLOGY (Evaluator-Optimizer Pattern):
1. GATHER: Collect relevant market data using available tools
2. ANALYZE: Identify patterns, trends, and risk factors
3. EVALUATE: Assess potential strategies against risk tolerance
4. REFINE: Iterate on recommendations based on constraints
5. PRESENT: Provide clear, actionable recommendations

Guidelines:
- Be data-driven with specific numbers and percentages
- Always consider downside risk alongside upside potential
- Provide confidence levels for your recommendations
- Suggest position sizes relative to portfolio value
- Never execute trades - only analyze and recommend

RISK FRAMEWORK:
- Conservative: Focus on capital preservation, lower volatility assets
- Moderate: Balanced risk/reward, diversified positions
- Aggressive: Higher risk tolerance, momentum plays acceptable

Available tools: get_market_data, get_portfolio, get_indicators`

// NewStrategist creates a market strategist sub-agent.
// This agent analyzes market data and portfolio positions to provide
// investment strategy recommendations following the evaluator-optimizer pattern.
func NewStrategist(eng *engine.Engine) *subagent.SubAgent {
	return subagent.NewSubAgent(eng, subagent.SubAgentConfig{
		Name:         "strategist",
		SystemPrompt: StrategistSystemPrompt,
		AvailableTools: []string{
			"get_market_data",
			"get_portfolio",
			"get_indicators",
		},
		MaxTurns:  8, // More turns for iterative analysis
		MaxTokens: 2048,
	})
}

// NewStrategistDelegationTool creates a delegation tool for the strategist.
// The main agent can delegate market analysis tasks to this specialist.
func NewStrategistDelegationTool(eng *engine.Engine) *subagent.DelegationTool {
	return subagent.NewDelegationTool(subagent.DelegationConfig{
		SubAgent:    NewStrategist(eng),
		ToolName:    "analyze_market",
		Description: "Delegate market analysis to the strategist specialist. Use this for investment strategy recommendations, market trend analysis, and portfolio optimization suggestions.",
		TaskFormatter: func(query string) string {
			return "Analyze the market and provide strategy recommendations for: " + query
		},
		QueryDescription: "The market analysis request (e.g., 'Should I increase my BTC position?', 'What's the outlook for ETH?')",
	})
}

// StrategistWithRiskProfile creates a strategist with a specific risk tolerance.
// This demonstrates the routing pattern - different configurations for different needs.
func StrategistWithRiskProfile(eng *engine.Engine, riskProfile string) *subagent.SubAgent {
	var riskPrompt string
	switch riskProfile {
	case "conservative":
		riskPrompt = `
RISK PROFILE: CONSERVATIVE
- Prioritize capital preservation above all
- Recommend only stable, low-volatility positions
- Suggest smaller position sizes (max 5% per trade)
- Emphasize diversification and hedging strategies
- Avoid momentum plays and speculative assets`
	case "aggressive":
		riskPrompt = `
RISK PROFILE: AGGRESSIVE
- Higher risk tolerance is acceptable
- Momentum plays and trend-following strategies welcomed
- Position sizes up to 20% per trade allowed
- Focus on asymmetric risk/reward opportunities
- Consider leveraged strategies when appropriate`
	default: // moderate
		riskPrompt = `
RISK PROFILE: MODERATE
- Balance risk and reward in all recommendations
- Standard position sizes (5-10% per trade)
- Mix of stable and growth-oriented positions
- Hedging recommended for concentrated positions
- Avoid excessive leverage`
	}

	return subagent.NewSubAgent(eng, subagent.SubAgentConfig{
		Name:         "strategist-" + riskProfile,
		SystemPrompt: StrategistSystemPrompt + riskPrompt,
		AvailableTools: []string{
			"get_market_data",
			"get_portfolio",
			"get_indicators",
		},
		MaxTurns:  8,
		MaxTokens: 2048,
	})
}
