// Package presets provides pre-configured sub-agents for common tasks.
package presets

import (
	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/engine"
	"github.com/becomeliminal/nim-go-sdk/subagent"
)

// OrchestratorSystemPrompt implements the orchestrator-workers pattern from
// Anthropic's "Building Effective Agents" guidance. The orchestrator dynamically
// breaks down tasks, delegates to worker agents, and synthesizes results.
const OrchestratorSystemPrompt = `You are a financial orchestrator agent.

Your role is to coordinate complex financial tasks by delegating to specialist agents
and synthesizing their results into actionable recommendations.

AVAILABLE SPECIALISTS:
- analyze_spending: Financial analyst for transaction analysis and budget insights
- research_recipient: User researcher for finding and verifying transfer recipients  
- analyze_market: Market strategist for investment analysis and trading recommendations

ORCHESTRATION METHODOLOGY:
1. DECOMPOSE: Break the user's request into subtasks that specialists can handle
2. DELEGATE: Route each subtask to the appropriate specialist agent
3. SYNTHESIZE: Combine specialist responses into a coherent answer
4. VALIDATE: Ensure the combined response fully addresses the user's needs

ROUTING GUIDELINES:
- For spending/budget questions → use analyze_spending
- For finding users/recipients → use research_recipient
- For market/investment questions → use analyze_market
- For complex queries → use multiple specialists sequentially

IMPORTANT:
- You coordinate but don't have direct data access - always delegate
- Wait for specialist responses before synthesizing
- Be explicit about which specialists you're consulting
- If a specialist fails, explain the limitation and offer alternatives

Example decomposition:
User: "Should I invest my savings or pay off debt first?"
→ analyze_spending (to understand current cash flow)
→ analyze_market (to assess investment opportunities)
→ Synthesize: Compare expected returns vs debt interest costs`

// Orchestrator represents a coordinating agent that delegates to specialists.
// It implements the orchestrator-workers pattern for complex multi-step tasks.
type Orchestrator struct {
	engine       *engine.Engine
	workers      map[string]*subagent.DelegationTool
	systemPrompt string
}

// OrchestratorConfig configures the orchestrator agent.
type OrchestratorConfig struct {
	// Engine is the agent execution engine.
	Engine *engine.Engine

	// Workers are the specialist agents available for delegation.
	// Map key is the tool name, value is the delegation tool.
	Workers map[string]*subagent.DelegationTool

	// SystemPromptOverride optionally overrides the default orchestrator prompt.
	SystemPromptOverride string
}

// NewOrchestrator creates an orchestrator agent with the default specialists.
func NewOrchestrator(eng *engine.Engine) *Orchestrator {
	workers := make(map[string]*subagent.DelegationTool)

	// Register default specialists
	workers["analyze_spending"] = NewAnalystDelegationTool(eng)
	workers["research_recipient"] = NewResearcherDelegationTool(eng)
	workers["analyze_market"] = NewStrategistDelegationTool(eng)

	return &Orchestrator{
		engine:       eng,
		workers:      workers,
		systemPrompt: OrchestratorSystemPrompt,
	}
}

// NewOrchestratorWithConfig creates an orchestrator with custom configuration.
func NewOrchestratorWithConfig(cfg OrchestratorConfig) *Orchestrator {
	prompt := OrchestratorSystemPrompt
	if cfg.SystemPromptOverride != "" {
		prompt = cfg.SystemPromptOverride
	}

	return &Orchestrator{
		engine:       cfg.Engine,
		workers:      cfg.Workers,
		systemPrompt: prompt,
	}
}

// AddWorker adds a specialist worker to the orchestrator.
func (o *Orchestrator) AddWorker(tool *subagent.DelegationTool) {
	o.workers[tool.Name()] = tool
}

// GetWorkerTools returns the delegation tools for registration with the engine.
func (o *Orchestrator) GetWorkerTools() []core.Tool {
	tools := make([]core.Tool, 0, len(o.workers))
	for _, w := range o.workers {
		tools = append(tools, w)
	}
	return tools
}

// SystemPrompt returns the orchestrator's system prompt.
func (o *Orchestrator) SystemPrompt() string {
	return o.systemPrompt
}

// WorkerNames returns the names of all available worker agents.
func (o *Orchestrator) WorkerNames() []string {
	names := make([]string, 0, len(o.workers))
	for name := range o.workers {
		names = append(names, name)
	}
	return names
}

// Capabilities returns the orchestrator's capabilities for use as a core.Agent.
func (o *Orchestrator) Capabilities() *core.Capabilities {
	return &core.Capabilities{
		CanRequestConfirmation: true, // Orchestrator CAN request confirmation
		AvailableTools:         o.WorkerNames(),
		Model:                  "claude-sonnet-4-20250514",
		MaxTokens:              4096,
		MaxTurns:               15, // More turns for coordination
		SystemPrompt:           o.systemPrompt,
	}
}
