// Trading-enabled Hackathon Starter: AI Financial Agent + Autonomous Trading
// Build intelligent financial tools with trading capabilities
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/executor"
	"github.com/becomeliminal/nim-go-sdk/server"
	"github.com/becomeliminal/nim-go-sdk/tools"
	"github.com/joho/godotenv"

	// Import our trading package
	"github.com/becomeliminal/nim-go-sdk/examples/hackathon-starter/trading"
)

// Global trading system (initialized when user sets budget)
var globalTradingSystem *trading.TradingSystem

func main() {
	// ============================================================================
	// CONFIGURATION
	// ============================================================================
	_ = godotenv.Load()

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	if anthropicKey == "" {
		log.Fatal("❌ ANTHROPIC_API_KEY environment variable is required")
	}

	liminalBaseURL := os.Getenv("LIMINAL_BASE_URL")
	if liminalBaseURL == "" {
		liminalBaseURL = "https://api.liminal.cash"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// ============================================================================
	// LIMINAL EXECUTOR SETUP
	// ============================================================================
	liminalExecutor := executor.NewHTTPExecutor(executor.HTTPExecutorConfig{
		BaseURL: liminalBaseURL,
	})
	log.Println("✅ Liminal API configured")

	// ============================================================================
	// TRADING SYSTEM SETUP (with default configuration)
	// ============================================================================
	// Initialize with default budget - user can reconfigure via chat
	defaultConfig := trading.GetDefaultConfig()
	globalTradingSystem = trading.NewTradingSystem(
		defaultConfig.DefaultBudget,
		defaultConfig.DefaultStopLossFloor,
		defaultConfig.DefaultRiskProfile,
	)
	log.Println("✅ Trading system initialized (default: $10 budget, $7 floor, conservative)")

	// ============================================================================
	// JOURNEY SCHEMA INITIALIZATION (Stabilize → Save → Invest flow)
	// ============================================================================
	if globalTradingSystem.DB != nil {
		if err := globalTradingSystem.DB.InitializeJourneySchema(); err != nil {
			log.Printf("⚠️ Failed to initialize journey schema: %v", err)
		} else {
			log.Println("✅ Journey schema initialized (user plans, budgets, savings rules, emergency funds)")
		}
	}
	seedDemoEmergencyFund(globalTradingSystem.DB)

	// ============================================================================
	// SQLITE CONVERSATION STORE (Persistent Chat History)
	// ============================================================================
	chatStore, err := NewSQLiteConversations("chat_history.db")
	if err != nil {
		log.Printf("⚠️ Failed to initialize SQLite chat store: %v (using in-memory)", err)
		chatStore = nil
	} else {
		log.Println("✅ SQLite chat history store initialized (chat_history.db)")
	}

	// ============================================================================
	// SERVER SETUP
	// ============================================================================
	serverConfig := server.Config{
		AnthropicKey:    anthropicKey,
		SystemPrompt:    tradingSystemPrompt,
		Model:           "claude-sonnet-4-20250514",
		MaxTokens:       4096,
		LiminalExecutor: liminalExecutor,
	}

	// Use SQLite store if available
	if chatStore != nil {
		serverConfig.Conversations = chatStore
	}

	srv, err := server.New(serverConfig)
	if err != nil {
		log.Fatal(err)
	}

	// ============================================================================
	// ADD LIMINAL BANKING TOOLS
	// ============================================================================
	srv.AddTools(tools.LiminalTools(liminalExecutor)...)
	log.Println("✅ Added 9 Liminal banking tools")
	if trading.DemoMode {
		srv.AddTools(trading.DemoLiminalTools()...)
		log.Println("✅ Added demo Liminal tools (mock balance, savings, transactions, profile)")
	}

	// ============================================================================
	// ADD TRADING TOOLS
	// ============================================================================
	// Add all trading tools (market data, indicators, portfolio)
	srv.AddTools(globalTradingSystem.GetAllTools()...)
	log.Println("✅ Added 6 trading tools (market data, indicators, portfolio)")

	// Add trading configuration tool
	srv.AddTool(createConfigureTradingTool())
	log.Println("✅ Added trading configuration tool")

	// Add trading budget allocation tool
	srv.AddTool(createAllocateTradingBudgetTool(liminalExecutor))
	log.Println("✅ Added trading budget allocation tool")

	// Add trading preferences tool
	srv.AddTool(createSetTradingPreferencesTool())
	log.Println("✅ Added trading preferences tool")

	// Add autonomous trading tools
	srv.AddTool(createStartAutoTradingTool(srv))
	srv.AddTool(createStopAutoTradingTool())
	srv.AddTool(createGetAutoTradingStatusTool())
	log.Println("✅ Added 3 autonomous trading tools")

	// Add withdrawal tool
	srv.AddTool(createWithdrawTradingProfitsTool(liminalExecutor))
	log.Println("✅ Added withdrawal tool")

	// ============================================================================
	// ADD HACKATHON INSIGHT TOOLS (Non-obvious insights from wallet data)
	// ============================================================================
	srv.AddTool(trading.CreateAnalyzeSpendingPatternsTool(liminalExecutor))
	srv.AddTool(trading.CreateSavingsOptimizerTool(liminalExecutor))
	srv.AddTool(trading.CreateTradingReadinessAssessmentTool(liminalExecutor, globalTradingSystem.Portfolio))
	srv.AddTool(trading.CreateFinancialHealthScoreTool(liminalExecutor))
	srv.AddTool(trading.CreateSmartBudgetRecommendationTool(liminalExecutor))
	srv.AddTool(trading.CreateBudgetPlannerTool(liminalExecutor))
	srv.AddTool(trading.CreateDetectSubscriptionsTool(liminalExecutor))
	srv.AddTool(trading.CreateGetSavingsAnalysisTool(liminalExecutor))
	srv.AddTool(trading.CreateExecuteSavingsSweepTool(liminalExecutor))
	log.Println("✅ Added 9 hackathon insight tools (spending patterns, savings optimizer, trading readiness, financial health, smart budget, budget planner, subscription detector, smart savings analysis, savings sweep)")

	// Add demo mode toggle tool
	srv.AddTool(createToggleDemoModeTool())
	log.Printf("✅ Added demo mode toggle tool (current: demo_mode=%t)", trading.DemoMode)

	// ============================================================================
	// ADD JOURNEY TOOLS (Stabilize → Save → Invest flow)
	// ============================================================================
	if globalTradingSystem.DB != nil {
		journeyTools := trading.GetJourneyTools(globalTradingSystem.DB, liminalExecutor)
		srv.AddTools(journeyTools...)
		log.Printf("✅ Added %d journey tools (onboarding, budget, savings rules, emergency fund, investment surplus, autopilot)", len(journeyTools))
	}

	// ============================================================================
	// ADD BINANCE CONNECTOR TOOLS (if configured)
	// ============================================================================
	binanceTools := globalTradingSystem.GetBinanceTools()
	if len(binanceTools) > 0 {
		srv.AddTools(binanceTools...)
		log.Printf("✅ Added %d Binance trading tools", len(binanceTools))
	}

	// ============================================================================
	// START SERVER
	// ============================================================================
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("🚀 Trading-Enabled Hackathon Starter Running")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📡 WebSocket endpoint: ws://localhost:%s/ws", port)
	log.Printf("💚 Health check: http://localhost:%s/health", port)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("📈 Trading Features:")
	log.Println("   • Market data: BTCUSDT, ETHUSDT, XAUUSD, EURUSD")
	log.Println("   • Technical analysis: RSI, MACD, Bollinger Bands")
	log.Println("   • Portfolio management with stop-loss protection")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("Ready for connections! Start your frontend with: cd frontend && npm run dev")
	log.Println()

	// Add conversations list API endpoint
	http.HandleFunc("/api/conversations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userID := "user" // Default user for demo

		if r.Method == "GET" {
			if chatStore == nil {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"conversations": []interface{}{},
				})
				return
			}

			conversations, err := chatStore.List(r.Context(), userID, 50)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"conversations": conversations,
			})
			return
		}

		// DELETE a specific conversation
		if r.Method == "DELETE" {
			convID := r.URL.Query().Get("id")
			if convID == "" {
				http.Error(w, "Missing conversation id", http.StatusBadRequest)
				return
			}

			if chatStore != nil {
				if err := chatStore.Delete(r.Context(), convID); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"deleted": convID,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	log.Printf("💬 Conversations API: http://localhost:%s/api/conversations", port)

	// Add demo mode API endpoint
	http.HandleFunc("/api/demo-mode", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == "GET" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"demo_mode": trading.DemoMode,
				"message":   getDemoModeMessage(),
			})
			return
		}

		if r.Method == "POST" {
			var params struct {
				Enable bool `json:"enable"`
			}
			if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			trading.DemoMode = params.Enable
			log.Printf("Demo mode changed to: %t", trading.DemoMode)

			json.NewEncoder(w).Encode(map[string]interface{}{
				"demo_mode": trading.DemoMode,
				"message":   getDemoModeMessage(),
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	log.Printf("🎮 Demo mode API: http://localhost:%s/api/demo-mode", port)

	// Add budget plan API endpoint for Budget Dashboard
	http.HandleFunc("/api/budget-plan", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse optional query parameters
		goal := r.URL.Query().Get("goal")
		if goal == "" {
			goal = "general_savings"
		}
		riskTolerance := r.URL.Query().Get("risk_tolerance")
		if riskTolerance == "" {
			riskTolerance = "moderate"
		}

		// Generate budget plan using demo or real data
		budgetPlan := trading.GenerateBudgetPlanForAPI(
			r.Context(),
			liminalExecutor,
			goal,
			riskTolerance,
		)

		json.NewEncoder(w).Encode(budgetPlan)
	})
	log.Printf("📊 Budget plan API: http://localhost:%s/api/budget-plan", port)

	// Add subscriptions API endpoint for Subscription Manager
	http.HandleFunc("/api/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Generate subscription analysis
		analysis := trading.GenerateSubscriptionAnalysisForAPI(r.Context(), liminalExecutor)
		json.NewEncoder(w).Encode(analysis)
	})
	log.Printf("📋 Subscriptions API: http://localhost:%s/api/subscriptions", port)

	// Add savings analysis API endpoint for Smart Savings
	http.HandleFunc("/api/savings-analysis", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse optional risk tolerance parameter
		riskTolerance := r.URL.Query().Get("risk_tolerance")
		if riskTolerance == "" {
			riskTolerance = "low"
		}

		// Generate savings analysis
		analysis := trading.GenerateSavingsAnalysisForAPI(r.Context(), liminalExecutor, riskTolerance)
		json.NewEncoder(w).Encode(analysis)
	})
	log.Printf("💰 Savings analysis API: http://localhost:%s/api/savings-analysis", port)

	// Add trading status HTTP endpoint for dashboard
	http.HandleFunc("/api/trading-status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		status := globalTradingSystem.Portfolio.GetStatus()
		json.NewEncoder(w).Encode(status)
	})
	log.Printf("📊 Trading status API: http://localhost:%s/api/trading-status", port)

	// Add journey status HTTP endpoint for dashboard
	http.HandleFunc("/api/journey-status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := "user" // Default user for demo

		if globalTradingSystem.DB == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Database not available",
			})
			return
		}

		status, err := globalTradingSystem.DB.GetJourneyStatus(userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(status)
	})
	log.Printf("🚀 Journey status API: http://localhost:%s/api/journey-status", port)

	// Add budget API endpoint (synced with journey database)
	http.HandleFunc("/api/budget", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userID := "user" // Default user for demo

		if globalTradingSystem.DB == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Database not available",
			})
			return
		}

		// GET - Retrieve budget
		if r.Method == "GET" {
			budget, err := globalTradingSystem.DB.GetBudget(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if budget == nil {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"exists":  false,
					"message": "No budget created yet. Create one via chat or the UI.",
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"exists":       true,
				"user_id":      budget.UserID,
				"period":       budget.Period,
				"start_date":   budget.StartDate,
				"envelopes":    budget.Envelopes,
				"buffer":       budget.Buffer,
				"total_budget": budget.TotalBudget,
				"insights":     budget.Insights,
				"created_at":   budget.CreatedAt,
				"updated_at":   budget.UpdatedAt,
			})
			return
		}

		// POST - Create or update budget
		if r.Method == "POST" {
			var req struct {
				TotalBudget float64 `json:"total_budget"`
				Period      string  `json:"period"`
				StartDate   int     `json:"start_date"`
				Buffer      float64 `json:"buffer"`
				Envelopes   []struct {
					Name      string  `json:"name"`
					Amount    float64 `json:"amount"`
					Guardrail string  `json:"guardrail"`
					Threshold float64 `json:"threshold"`
					Category  string  `json:"category"`
					Color     string  `json:"color"`
				} `json:"envelopes"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			// Default values
			if req.Period == "" {
				req.Period = "monthly"
			}
			if req.StartDate == 0 {
				req.StartDate = 1
			}

			// Convert envelopes
			envelopes := make([]trading.Envelope, len(req.Envelopes))
			for i, e := range req.Envelopes {
				envelopes[i] = trading.Envelope{
					Name:      e.Name,
					Amount:    e.Amount,
					Guardrail: e.Guardrail,
					Threshold: e.Threshold,
					Category:  e.Category,
					Color:     e.Color,
				}
			}

			// If no envelopes provided, create defaults based on 50/20/20/10 rule
			if len(envelopes) == 0 && req.TotalBudget > 0 {
				envelopes = []trading.Envelope{
					{Name: "Needs", Amount: req.TotalBudget * 0.50, Guardrail: "hard", Threshold: 1.0, Category: "needs", Color: "#007AFF"},
					{Name: "Wants", Amount: req.TotalBudget * 0.20, Guardrail: "soft", Threshold: 0.8, Category: "wants", Color: "#AF52DE"},
					{Name: "Bills", Amount: req.TotalBudget * 0.20, Guardrail: "auto_pay", Threshold: 1.0, Category: "bills", Color: "#FF9500"},
					{Name: "Goals", Amount: req.TotalBudget * 0.10, Guardrail: "protected", Threshold: 1.0, Category: "savings", Color: "#34C759"},
				}
			}

			budget := &trading.Budget{
				UserID:      userID,
				Period:      req.Period,
				StartDate:   req.StartDate,
				Envelopes:   envelopes,
				Buffer:      req.Buffer,
				TotalBudget: req.TotalBudget,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := globalTradingSystem.DB.SaveBudget(budget); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":      true,
				"message":      "Budget saved successfully",
				"total_budget": budget.TotalBudget,
				"envelopes":    budget.Envelopes,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	log.Printf("💵 Budget API: http://localhost:%s/api/budget", port)

	// Add savings rules API endpoint (synced with journey database)
	http.HandleFunc("/api/savings-rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userID := "user"

		if globalTradingSystem.DB == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "Database not available"})
			return
		}

		// GET - List all savings rules
		if r.Method == "GET" {
			rules, err := globalTradingSystem.DB.GetSavingsRules(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
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
					"total_saved": r.TotalSaved,
					"run_count":   r.RunCount,
					"created_at":  r.CreatedAt,
				}
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":     true,
				"rules":       rulesList,
				"total_rules": len(rules),
				"total_saved": totalSaved,
			})
			return
		}

		// POST - Create a new savings rule
		if r.Method == "POST" {
			var req struct {
				Name        string  `json:"name"`
				Type        string  `json:"type"`
				Trigger     string  `json:"trigger"`
				Action      string  `json:"action"`
				Amount      float64 `json:"amount"`
				Percentage  float64 `json:"percentage"`
				Destination string  `json:"destination"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			rule := &trading.SavingsRule{
				ID:          fmt.Sprintf("rule_%d", time.Now().UnixNano()),
				UserID:      userID,
				Name:        req.Name,
				Type:        req.Type,
				Trigger:     req.Trigger,
				Action:      req.Action,
				Amount:      req.Amount,
				Percentage:  req.Percentage,
				Destination: req.Destination,
				Active:      true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := globalTradingSystem.DB.SaveSavingsRule(rule); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Savings rule created",
				"rule":    rule,
			})
			return
		}

		// DELETE - Delete a savings rule
		if r.Method == "DELETE" {
			ruleID := r.URL.Query().Get("id")
			if ruleID == "" {
				http.Error(w, "Missing rule id", http.StatusBadRequest)
				return
			}

			// For now, just return success (would need to add delete method to DB)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Rule deleted",
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	log.Printf("📋 Savings Rules API: http://localhost:%s/api/savings-rules", port)

	// Add emergency fund API endpoint
	http.HandleFunc("/api/emergency-fund", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		userID := "user"

		if globalTradingSystem.DB == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "Database not available"})
			return
		}

		// GET - Get emergency fund status
		if r.Method == "GET" {
			ef, err := globalTradingSystem.DB.GetEmergencyFund(userID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if ef == nil {
				// Return default structure if not set up
				json.NewEncoder(w).Encode(map[string]interface{}{
					"exists":         false,
					"current":        0,
					"stage_1_target": 0,
					"stage_2_target": 0,
					"stage_3_target": 0,
					"current_stage":  0,
					"message":        "Emergency fund not set up. Complete onboarding first.",
				})
				return
			}

			// Calculate stage progress
			stage1Progress := 0.0
			stage2Progress := 0.0
			stage3Progress := 0.0
			if ef.Stage1Target > 0 {
				stage1Progress = (ef.Current / ef.Stage1Target) * 100
			}
			if ef.Stage2Target > 0 {
				stage2Progress = (ef.Current / ef.Stage2Target) * 100
			}
			if ef.Stage3Target > 0 {
				stage3Progress = (ef.Current / ef.Stage3Target) * 100
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"exists":             true,
				"current":            ef.Current,
				"stage_1_target":     ef.Stage1Target,
				"stage_2_target":     ef.Stage2Target,
				"stage_3_target":     ef.Stage3Target,
				"current_stage":      ef.CurrentStage,
				"monthly_expenses":   ef.MonthlyExpenses,
				"stage_1_progress":   stage1Progress,
				"stage_2_progress":   stage2Progress,
				"stage_3_progress":   stage3Progress,
				"investing_unlocked": ef.CurrentStage >= 1,
				"stages": []map[string]interface{}{
					{
						"stage":       1,
						"name":        "2 Weeks",
						"target":      ef.Stage1Target,
						"progress":    math.Min(stage1Progress, 100),
						"complete":    ef.Current >= ef.Stage1Target,
						"description": "2 weeks of expenses - unlocks investing",
					},
					{
						"stage":       2,
						"name":        "1 Month",
						"target":      ef.Stage2Target,
						"progress":    math.Min(stage2Progress, 100),
						"complete":    ef.Current >= ef.Stage2Target,
						"description": "1 month of expenses - solid foundation",
					},
					{
						"stage":       3,
						"name":        "3 Months",
						"target":      ef.Stage3Target,
						"progress":    math.Min(stage3Progress, 100),
						"complete":    ef.Current >= ef.Stage3Target,
						"description": "3 months of expenses - fully funded",
					},
				},
			})
			return
		}

		// POST - Update emergency fund current amount (for manual updates)
		if r.Method == "POST" {
			var req struct {
				Current float64 `json:"current"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}

			ef, _ := globalTradingSystem.DB.GetEmergencyFund(userID)
			if ef == nil {
				http.Error(w, "Emergency fund not set up", http.StatusBadRequest)
				return
			}

			ef.Current = req.Current
			// Update stage based on current amount
			if ef.Current >= ef.Stage3Target {
				ef.CurrentStage = 3
			} else if ef.Current >= ef.Stage2Target {
				ef.CurrentStage = 2
			} else if ef.Current >= ef.Stage1Target {
				ef.CurrentStage = 1
			} else {
				ef.CurrentStage = 0
			}

			if err := globalTradingSystem.DB.SaveEmergencyFund(ef); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":       true,
				"current":       ef.Current,
				"current_stage": ef.CurrentStage,
			})
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
	log.Printf("🛡️ Emergency Fund API: http://localhost:%s/api/emergency-fund", port)

	// Add withdrawal API endpoint
	http.HandleFunc("/api/withdraw_trading_profits", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var params struct {
			Amount   float64 `json:"amount"`
			Strategy string  `json:"strategy"`
		}
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		result, err := executeWithdrawal(context.Background(), liminalExecutor, params.Amount, params.Strategy)
		if err != nil {
			// Determine if it's a validation error or internal error
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(result)
	})
	log.Printf("💸 Withdrawal API: http://localhost:%s/api/withdraw_trading_profits", port)

	// Add auth proxy endpoint for OTP
	http.HandleFunc("/auth/v1/otp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Proxy to real Liminal API
		client := &http.Client{Timeout: 10 * time.Second}
		body, _ := io.ReadAll(r.Body)
		req, err := http.NewRequest("POST", liminalBaseURL+"/auth/v1/otp", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	// Add auth proxy endpoint for verify
	http.HandleFunc("/auth/v1/verify", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Proxy to real Liminal API
		client := &http.Client{Timeout: 10 * time.Second}
		body, _ := io.ReadAll(r.Body)
		req, err := http.NewRequest("POST", liminalBaseURL+"/auth/v1/verify", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})

	// Add auth proxy endpoint for token refresh
	http.HandleFunc("/auth/v1/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Proxy to real Liminal API
		client := &http.Client{Timeout: 10 * time.Second}
		body, _ := io.ReadAll(r.Body)
		req, err := http.NewRequest("POST", liminalBaseURL+"/auth/v1/token", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
	log.Println("✅ Auth proxy endpoints configured")

	if err := srv.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// ============================================================================
// SYSTEM PROMPT (Extended with Trading)
// ============================================================================

const tradingSystemPrompt = `You are Nim, a friendly AI financial assistant that guides users through a proven financial journey:

**STABILIZE → SAVE → INVEST**

This philosophy ensures users build a solid foundation before taking on investment risk.

================================================================================
YOUR CORE PHILOSOPHY
================================================================================

1. **Stabilize First**: Help users understand their cash flow, control spending, eliminate wasteful subscriptions
2. **Save Next**: Build an emergency fund through automated savings rules before any investing
3. **Invest Last**: Only after Stage 1 emergency fund (2 weeks expenses) is complete

CONVERSATIONAL STYLE:
- Be warm, friendly, and supportive - financial journeys can be stressful
- Celebrate small wins and progress
- Use casual language but stay professional about money
- Guide users through the journey step by step
- Always check their journey status before making recommendations

================================================================================
THE 6-STEP JOURNEY
================================================================================

**Step 1: Chat with Nim (Onboarding)**
Tools: get_user_plan, create_user_plan, get_journey_status

When a new user arrives or asks "help me get started":
1. Use get_user_plan to check if they've completed onboarding
2. If not, gather their goals through conversation:
   - Monthly savings target
   - Income frequency and amount
   - Risk tolerance
   - Automation preferences (auto-sweep, round-ups)
3. Use create_user_plan to save their plan

**Step 2: Budget Planner**
Tools: create_budget, get_budget_status

After onboarding, help create an envelope-based budget:
1. Use create_budget with their income
2. Default: 50% needs, 20% wants, 20% bills, 10% goals
3. Set guardrails (soft alerts vs hard stops)

**Step 3: Subscriptions**
Tools: detect_subscriptions

Analyze their recurring payments:
- Find forgotten/unused subscriptions
- Detect price increases
- Identify duplicates
- Suggest cancellations

**Step 4: Smart Savings**
Tools: create_savings_rule, get_savings_rules, get_emergency_fund_status

Set up automated savings:
1. Payday sweep: Move fixed amount after income
2. Round-ups: Save spare change
3. Under-budget sweep: Save portion of unspent budget

Track emergency fund progress through 3 stages:
- Stage 1: 2 weeks expenses (UNLOCKS INVESTING)
- Stage 2: 1 month expenses
- Stage 3: 3-6 months expenses

**Step 5: Trading Terminal (GATED)**
Tools: get_investment_surplus, allocate_trading_budget, get_trading_status

⚠️ CRITICAL: Investing is LOCKED until Stage 1 emergency fund is complete!

When user asks about trading:
1. ALWAYS check get_emergency_fund_status first
2. If Stage 1 incomplete, explain why they need to wait
3. If Stage 1 complete, use get_investment_surplus to see available funds
4. Only invest SURPLUS - never touch emergency fund or budget

If user says "trade $X" or "invest $X":
1. get_emergency_fund_status
2. If Stage 1 complete, call allocate_trading_budget with amount=X
   - Default stop_loss_floor to 80% of amount if not provided
   - Default risk_profile to moderate if not provided
3. After allocation, call get_trading_status

**Step 6: Autopilot**
Tools: get_weekly_digest, approve_pending_action

Once set up, users get weekly digests with:
- Budget status
- Savings rules executed
- Emergency fund progress
- Pending actions to approve

================================================================================
BANKING CAPABILITIES
================================================================================
- get_balance: Check wallet balance
- get_savings_balance: Check savings and APY
- get_vault_rates: View available rates
- get_transactions: View history
- get_profile: User info
- send_money: Transfer funds (requires confirmation)
- deposit_savings: Move to savings (requires confirmation)
- withdraw_savings: Move from savings (requires confirmation)

================================================================================
INSIGHT TOOLS
================================================================================
- analyze_spending_patterns: Find trends and anomalies
- optimize_savings: Find idle cash, compare vault rates
- calculate_financial_health: Comprehensive health score
- get_smart_budget: AI-generated budget recommendations
- create_budget_plan: Full wealth forecasting
- get_savings_analysis: Detailed savings optimization
- execute_savings_sweep: Move idle cash to savings (requires confirmation)

================================================================================
TRADING TOOLS (Use only after emergency fund Stage 1)
================================================================================
- get_market_price: Current prices (BTCUSDT, ETHUSDT, XAUUSD, EURUSD)
- get_candles: Historical data
- calc_indicators: RSI, MACD, SMA, Bollinger Bands
- get_trading_status: Portfolio and positions
- open_trade / close_trade: Execute trades (requires confirmation)
- set_trading_preferences: Configure risk and style
- start_auto_trading / stop_auto_trading: Autonomous trading
- get_auto_trading_status: Monitor auto-trading

================================================================================
CRITICAL RULES
================================================================================

1. **Always check journey status first**: Use get_journey_status or get_user_plan before making recommendations
2. **Gate investing behind emergency fund**: NEVER suggest trading unless Stage 1 is complete
3. **Celebrate progress**: When users complete a stage, acknowledge it!
4. **Guide, don't push**: Let users move at their own pace through the journey
5. **Protect their foundation**: Emergency fund and budget envelopes are sacred

================================================================================
EXAMPLE INTERACTIONS
================================================================================

User: "I want to start investing"
→ 1) get_emergency_fund_status 2) If Stage 1 incomplete: "Let's build your emergency fund first - you're £X away from unlocking investing!"

User: "Help me get started"
→ 1) get_user_plan 2) If no plan: Start onboarding conversation 3) If plan exists: get_journey_status and suggest next step

User: "How am I doing?"
→ 1) get_journey_status 2) Show progress through steps 3) Highlight wins and next actions

User: "Set up automatic savings"
→ 1) create_savings_rule with payday sweep or round-ups 2) Explain how it will help reach emergency fund

User: "What's my budget status?"
→ 1) get_budget_status 2) Show envelope usage and any alerts

User: "I'm ready to invest"
→ 1) get_emergency_fund_status 2) If Stage 1 complete: get_investment_surplus 3) Recommend core (80%) vs explore (20%) allocation

User: "trade $50"
→ 1) get_emergency_fund_status 2) If Stage 1 complete: allocate_trading_budget (amount=50, stop_loss_floor=40, risk_profile=moderate) 3) get_trading_status

Remember: You're guiding users on a JOURNEY. Meet them where they are, celebrate their progress, and always protect their financial foundation!`

// ============================================================================
// DEMO MODE HELPERS
// ============================================================================

func getDemoModeMessage() string {
	if trading.DemoMode {
		return "Demo mode ENABLED - using mock transaction data for testing"
	}
	return "Demo mode DISABLED - using real Liminal API data"
}

func seedDemoEmergencyFund(db *trading.Database) {
	if db == nil || !trading.DemoMode {
		return
	}

	userID := "user"
	if existing, err := db.GetEmergencyFund(userID); err != nil {
		log.Printf("⚠️ Failed to read emergency fund status: %v", err)
	} else if existing != nil && existing.CurrentStage >= 1 {
		return
	}

	demoData, err := trading.LoadDemoDataFromFile()
	if err != nil {
		log.Printf("⚠️ Failed to load demo data for emergency fund seeding: %v", err)
		return
	}

	monthlyExpenses := demoData.EmergencyFund.MonthlyExpenses
	if monthlyExpenses == 0 {
		if demoData.MonthlyBudgets.TotalBudget > 0 {
			monthlyExpenses = demoData.MonthlyBudgets.TotalBudget
		} else if demoData.MonthlyBudgets.TotalIncome > 0 {
			monthlyExpenses = demoData.MonthlyBudgets.TotalIncome
		}
	}

	stage1Target := demoData.EmergencyFund.Stage1Target
	stage2Target := demoData.EmergencyFund.Stage2Target
	stage3Target := demoData.EmergencyFund.Stage3Target
	if stage1Target == 0 || stage2Target == 0 || stage3Target == 0 {
		if monthlyExpenses > 0 {
			stage1Target, stage2Target, stage3Target = trading.CalculateEmergencyFundTargets(monthlyExpenses)
		}
	}

	if stage1Target == 0 {
		log.Printf("⚠️ Demo emergency fund targets unavailable; skipping seed")
		return
	}

	current := demoData.EmergencyFund.Current
	if current == 0 {
		current = stage1Target
	}

	currentStage := demoData.EmergencyFund.CurrentStage
	if currentStage == 0 {
		switch {
		case current >= stage3Target:
			currentStage = 3
		case current >= stage2Target:
			currentStage = 2
		case current >= stage1Target:
			currentStage = 1
		default:
			currentStage = 0
		}
	}

	ef := &trading.EmergencyFund{
		UserID:          userID,
		Current:         current,
		Stage1Target:    stage1Target,
		Stage2Target:    stage2Target,
		Stage3Target:    stage3Target,
		CurrentStage:    currentStage,
		MonthlyExpenses: monthlyExpenses,
		UpdatedAt:       time.Now(),
	}

	if err := db.SaveEmergencyFund(ef); err != nil {
		log.Printf("⚠️ Failed to seed demo emergency fund: %v", err)
		return
	}

	log.Printf("✅ Seeded demo emergency fund (stage %d)", currentStage)
}

// ============================================================================
// DEMO MODE TOGGLE TOOL
// ============================================================================

func createToggleDemoModeTool() core.Tool {
	return tools.New("toggle_demo_mode").
		Description("Toggle demo mode on/off. When demo mode is ON, the system uses mock transaction data for testing. When OFF, it uses real Liminal API data.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"enable": tools.BooleanProperty("Set to true to enable demo mode, false to disable"),
		}, "enable")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Enable bool `json:"enable"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			trading.DemoMode = params.Enable

			status := "DISABLED (using real Liminal data)"
			if trading.DemoMode {
				status = "ENABLED (using mock data for testing)"
			}

			return map[string]interface{}{
				"success":   true,
				"demo_mode": trading.DemoMode,
				"message":   fmt.Sprintf("Demo mode is now %s", status),
			}, nil
		}).
		Build()
}

// ============================================================================
// CONFIGURE TRADING TOOL
// ============================================================================

func createConfigureTradingTool() core.Tool {
	return tools.New("configure_trading").
		Description("Configure the trading system with budget, stop-loss floor, and risk profile. Call this when user wants to start trading with specific parameters.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"budget":          tools.StringProperty("Total trading budget in USD (e.g., '10')"),
			"stop_loss_floor": tools.StringProperty("Minimum portfolio value - trading stops if it hits this (e.g., '7')"),
			"risk_profile":    tools.StringProperty("Risk profile: 'conservative', 'moderate', or 'aggressive'"),
		}, "budget", "stop_loss_floor")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Budget        float64 `json:"budget"`
				StopLossFloor float64 `json:"stop_loss_floor"`
				RiskProfile   string  `json:"risk_profile"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			if params.Budget <= 0 {
				return map[string]interface{}{
					"success": false,
					"error":   "Budget must be greater than 0",
				}, nil
			}

			if params.StopLossFloor >= params.Budget {
				return map[string]interface{}{
					"success": false,
					"error":   "Stop-loss floor must be less than budget",
				}, nil
			}

			if params.RiskProfile == "" {
				params.RiskProfile = "conservative"
			}

			// Validate risk profile
			validProfiles := []string{"conservative", "moderate", "aggressive"}
			isValid := false
			for _, p := range validProfiles {
				if p == params.RiskProfile {
					isValid = true
					break
				}
			}
			if !isValid {
				return map[string]interface{}{
					"success": false,
					"error":   "Invalid risk profile. Use: conservative, moderate, or aggressive",
				}, nil
			}

			// Reconfigure the trading system
			globalTradingSystem = trading.NewTradingSystem(
				params.Budget,
				params.StopLossFloor,
				params.RiskProfile,
			)

			return map[string]interface{}{
				"success":         true,
				"budget":          fmt.Sprintf("$%.2f", params.Budget),
				"stop_loss_floor": fmt.Sprintf("$%.2f", params.StopLossFloor),
				"risk_profile":    params.RiskProfile,
				"message":         fmt.Sprintf("Trading configured! Budget: $%.2f, Floor: $%.2f, Profile: %s", params.Budget, params.StopLossFloor, params.RiskProfile),
			}, nil
		}).
		Build()
}

// ============================================================================
// ALLOCATE TRADING BUDGET TOOL
// ============================================================================

func createAllocateTradingBudgetTool(liminalExecutor core.ToolExecutor) core.Tool {
	return tools.New("allocate_trading_budget").
		Description("Allocate funds from Liminal wallet to trading account. ⚠️ DEMO MODE: This simulates fund transfer - no real money moves. Transfers are instant for testing purposes.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"amount":          tools.NumberProperty("Amount to allocate for trading in USD (e.g., 50.00)"),
			"stop_loss_floor": tools.NumberProperty("Minimum portfolio value - trading stops if it drops below this (e.g., 40.00)"),
			"risk_profile":    tools.StringEnumProperty("Risk profile", "conservative", "moderate", "aggressive"),
		}, "amount", "stop_loss_floor", "risk_profile")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Amount        float64 `json:"amount"`
				StopLossFloor float64 `json:"stop_loss_floor"`
				RiskProfile   string  `json:"risk_profile"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			// Validate inputs
			if params.Amount <= 0 {
				return map[string]interface{}{
					"success": false,
					"error":   "Amount must be greater than 0",
				}, nil
			}

			if params.StopLossFloor >= params.Amount {
				return map[string]interface{}{
					"success": false,
					"error":   "Stop-loss floor must be less than the allocated amount",
				}, nil
			}

			// Validate risk profile
			validProfiles := []string{"conservative", "moderate", "aggressive"}
			isValid := false
			for _, p := range validProfiles {
				if p == params.RiskProfile {
					isValid = true
					break
				}
			}
			if !isValid {
				return map[string]interface{}{
					"success": false,
					"error":   "Invalid risk profile. Use: conservative, moderate, or aggressive",
				}, nil
			}

			// Create fund bridge service
			fundBridge := trading.NewFundBridge(liminalExecutor, globalTradingSystem.Binance, globalTradingSystem.DB)

			// Get user ID from context (this would come from the request context)
			userID := "user123" // TODO: Get from actual request context

			// Transfer funds from Liminal to trading account
			if err := fundBridge.TransferToTrading(ctx, userID, params.Amount); err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("Failed to transfer funds: %v", err),
				}, nil
			}

			// Configure the trading system with the allocated budget
			globalTradingSystem = trading.NewTradingSystem(
				params.Amount,
				params.StopLossFloor,
				params.RiskProfile,
			)

			// Save allocation to database
			allocation := &trading.TradingAllocation{
				UserID:        userID,
				Amount:        params.Amount,
				StopLossFloor: params.StopLossFloor,
				RiskProfile:   params.RiskProfile,
				Status:        "active",
				AllocatedAt:   time.Now(),
			}

			if globalTradingSystem.DB != nil {
				if err := globalTradingSystem.DB.SaveAllocation(allocation); err != nil {
					log.Printf("⚠️ Failed to save allocation: %v", err)
				}
			}

			return map[string]interface{}{
				"success":         true,
				"amount":          fmt.Sprintf("$%.2f", params.Amount),
				"stop_loss_floor": fmt.Sprintf("$%.2f", params.StopLossFloor),
				"risk_profile":    params.RiskProfile,
				"message":         fmt.Sprintf("✅ Successfully allocated $%.2f for trading! Funds transferred from Liminal to Binance. Trading system configured with stop-loss floor at $%.2f and %s risk profile.", params.Amount, params.StopLossFloor, params.RiskProfile),
			}, nil
		}).
		Build()
}

// ============================================================================
// SET TRADING PREFERENCES TOOL
// ============================================================================

func createSetTradingPreferencesTool() core.Tool {
	return tools.New("set_trading_preferences").
		Description("Set user's trading preferences including asset focus, trading style, risk tolerance, and profit targets. Call this when user wants to customize their trading approach.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"assets":           tools.StringProperty("Comma-separated list of assets to trade (e.g., 'BTCUSDT,ETHUSDT')"),
			"style":            tools.StringEnumProperty("Trading style", "hft", "day_trading", "swing", "hold"),
			"risk_profile":     tools.StringEnumProperty("Risk profile", "conservative", "moderate", "aggressive"),
			"profit_target":    tools.NumberProperty("Profit target as decimal (e.g., 0.10 for 10%)"),
			"max_loss_percent": tools.NumberProperty("Maximum loss percentage as decimal (e.g., 0.05 for 5%)"),
			"auto_trade":       tools.BooleanProperty("Enable autonomous trading"),
		}, "assets", "style", "risk_profile")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Assets         string  `json:"assets"`
				Style          string  `json:"style"`
				RiskProfile    string  `json:"risk_profile"`
				ProfitTarget   float64 `json:"profit_target"`
				MaxLossPercent float64 `json:"max_loss_percent"`
				AutoTrade      bool    `json:"auto_trade"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			// Parse assets from comma-separated string
			var assets []string
			if params.Assets != "" {
				// Simple split by comma and trim spaces
				parts := strings.Split(params.Assets, ",")
				for _, part := range parts {
					asset := strings.TrimSpace(part)
					if asset != "" {
						assets = append(assets, asset)
					}
				}
			}

			// Create preferences object
			prefs := &trading.TradingPreferences{
				UserID:         "user123", // TODO: Get from actual request context
				Assets:         assets,
				Style:          params.Style,
				RiskProfile:    params.RiskProfile,
				ProfitTarget:   params.ProfitTarget,
				MaxLossPercent: params.MaxLossPercent,
				AutoTrade:      params.AutoTrade,
				UpdatedAt:      time.Now(),
			}

			// Set created_at if this is a new preference
			if existingPrefs, _ := globalTradingSystem.DB.GetPreferences(prefs.UserID); existingPrefs == nil {
				prefs.CreatedAt = time.Now()
			} else {
				prefs.CreatedAt = existingPrefs.CreatedAt
			}

			// Validate preferences
			if err := prefs.Validate(); err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("Invalid preferences: %v", err),
				}, nil
			}

			// Save to database
			if globalTradingSystem.DB != nil {
				if err := globalTradingSystem.DB.SavePreferences(prefs); err != nil {
					log.Printf("⚠️ Failed to save preferences: %v", err)
					return map[string]interface{}{
						"success": false,
						"error":   "Failed to save preferences to database",
					}, nil
				}
			}

			// Update global trading system preferences if it exists
			if globalTradingSystem != nil {
				// Note: In a full implementation, this would update the trading system's behavior
				log.Printf("✅ Updated trading preferences for user %s", prefs.UserID)
			}

			return map[string]interface{}{
				"success":          true,
				"assets":           prefs.Assets,
				"style":            prefs.Style,
				"risk_profile":     prefs.RiskProfile,
				"profit_target":    fmt.Sprintf("%.1f%%", prefs.ProfitTarget*100),
				"max_loss_percent": fmt.Sprintf("%.1f%%", prefs.MaxLossPercent*100),
				"auto_trade":       prefs.AutoTrade,
				"message":          fmt.Sprintf("✅ Trading preferences updated! Focus: %v, Style: %s, Risk: %s, Auto-trade: %t", prefs.Assets, prefs.Style, prefs.RiskProfile, prefs.AutoTrade),
			}, nil
		}).
		Build()
}

// ============================================================================
// SPENDING ANALYZER (from original)
// ============================================================================

func createSpendingAnalyzerTool(liminalExecutor core.ToolExecutor) core.Tool {
	return tools.New("analyze_spending").
		Description("Analyze the user's spending patterns over a specified time period.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"days": tools.IntegerProperty("Number of days to analyze (default: 30)"),
		})).
		Handler(func(ctx context.Context, toolParams *core.ToolParams) (*core.ToolResult, error) {
			var params struct {
				Days int `json:"days"`
			}
			if err := json.Unmarshal(toolParams.Input, &params); err != nil {
				return &core.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("invalid input: %v", err),
				}, nil
			}

			if params.Days == 0 {
				params.Days = 30
			}

			txRequest := map[string]interface{}{
				"limit": 100,
			}
			txRequestJSON, _ := json.Marshal(txRequest)

			txResponse, err := liminalExecutor.Execute(ctx, &core.ExecuteRequest{
				UserID:    toolParams.UserID,
				Tool:      "get_transactions",
				Input:     txRequestJSON,
				RequestID: toolParams.RequestID,
			})
			if err != nil {
				return &core.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("failed to fetch transactions: %v", err),
				}, nil
			}

			if !txResponse.Success {
				return &core.ToolResult{
					Success: false,
					Error:   fmt.Sprintf("transaction fetch failed: %s", txResponse.Error),
				}, nil
			}

			var transactions []map[string]interface{}
			var txData map[string]interface{}
			if err := json.Unmarshal(txResponse.Data, &txData); err == nil {
				if txArray, ok := txData["transactions"].([]interface{}); ok {
					for _, tx := range txArray {
						if txMap, ok := tx.(map[string]interface{}); ok {
							transactions = append(transactions, txMap)
						}
					}
				}
			}

			analysis := analyzeTransactions(transactions, params.Days)

			result := map[string]interface{}{
				"period_days":        params.Days,
				"total_transactions": len(transactions),
				"analysis":           analysis,
				"generated_at":       time.Now().Format(time.RFC3339),
			}

			return &core.ToolResult{
				Success: true,
				Data:    result,
			}, nil
		}).
		Build()
}

func analyzeTransactions(transactions []map[string]interface{}, days int) map[string]interface{} {
	if len(transactions) == 0 {
		return map[string]interface{}{
			"summary": "No transactions found in the specified period",
		}
	}

	var totalSpent, totalReceived float64
	var spendCount, receiveCount int

	for _, tx := range transactions {
		txType, _ := tx["type"].(string)
		amount, _ := tx["amount"].(float64)

		switch txType {
		case "send":
			totalSpent += amount
			spendCount++
		case "receive":
			totalReceived += amount
			receiveCount++
		}
	}

	avgDailySpend := totalSpent / float64(days)

	return map[string]interface{}{
		"total_spent":     fmt.Sprintf("%.2f", totalSpent),
		"total_received":  fmt.Sprintf("%.2f", totalReceived),
		"spend_count":     spendCount,
		"receive_count":   receiveCount,
		"avg_daily_spend": fmt.Sprintf("%.2f", avgDailySpend),
		"velocity":        calculateVelocity(spendCount, days),
	}
}

func calculateVelocity(transactionCount, days int) string {
	txPerWeek := float64(transactionCount) / float64(days) * 7

	switch {
	case txPerWeek < 2:
		return "low"
	case txPerWeek < 7:
		return "moderate"
	default:
		return "high"
	}
}

// ============================================================================
// AUTONOMOUS TRADING TOOLS
// ============================================================================

func createStartAutoTradingTool(srv *server.Server) core.Tool {
	return tools.New("start_auto_trading").
		Description("Start autonomous trading based on user's trading preferences. The system will analyze markets and execute trades automatically. Requires preferences to be set first with set_trading_preferences (with auto_trade=true).").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			// Get user preferences from database
			userID := "user123" // TODO: Get from actual request context

			if globalTradingSystem.DB == nil {
				return map[string]interface{}{
					"success": false,
					"error":   "Database not available",
				}, nil
			}

			prefs, err := globalTradingSystem.DB.GetPreferences(userID)
			if err != nil || prefs == nil {
				return map[string]interface{}{
					"success": false,
					"error":   "No trading preferences found. Use set_trading_preferences first.",
				}, nil
			}

			if !prefs.AutoTrade {
				return map[string]interface{}{
					"success": false,
					"error":   "Auto-trading is disabled in preferences. Set auto_trade=true first.",
				}, nil
			}

			// Check if allocation exists
			allocation, err := globalTradingSystem.DB.GetAllocation(userID)
			if err != nil || allocation == nil || allocation.Status != "active" {
				return map[string]interface{}{
					"success": false,
					"error":   "No active trading allocation. Use allocate_trading_budget first.",
				}, nil
			}

			// Create or update AutoTrader
			if globalTradingSystem.AutoTrader == nil {
				globalTradingSystem.AutoTrader = trading.NewAutoTrader(globalTradingSystem, prefs)
			} else {
				globalTradingSystem.AutoTrader.UpdatePreferences(prefs)
			}

			// Setup notification callback
			globalTradingSystem.AutoTrader.Notify = func(event string, data interface{}) {
				// Broadcast to user (assuming hardcoded user123 for now, or use prefs.UserID if available)
				userID := "user" // Hackathon default
				if prefs.UserID != "" {
					userID = prefs.UserID
				}

				srv.Broadcast(userID, server.ServerMessage{
					Type:    "notification",
					Content: event,
					Data:    data,
				})
			}

			// Start auto-trading
			if err := globalTradingSystem.AutoTrader.Start(ctx); err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"success":      true,
				"message":      "🤖 Auto-trading started!",
				"assets":       prefs.Assets,
				"style":        prefs.Style,
				"risk_profile": prefs.RiskProfile,
				"interval":     globalTradingSystem.AutoTrader.Interval.String(),
				"auto_trade":   true,
			}, nil
		}).
		Build()
}

func createStopAutoTradingTool() core.Tool {
	return tools.New("stop_auto_trading").
		Description("Stop autonomous trading. Optionally close all open positions.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"close_positions": tools.BooleanProperty("If true, close all open positions when stopping (default: false)"),
		})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				ClosePositions bool `json:"close_positions"`
			}
			_ = json.Unmarshal(input, &params)

			if globalTradingSystem.AutoTrader == nil {
				return map[string]interface{}{
					"success": false,
					"error":   "Auto-trading was never started.",
				}, nil
			}

			// Get final status before stopping
			status := globalTradingSystem.AutoTrader.GetStatus()

			// Stop auto-trading
			if err := globalTradingSystem.AutoTrader.Stop(); err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			// Close positions if requested
			var closedTrades []trading.Trade
			if params.ClosePositions {
				closedTrades = globalTradingSystem.Portfolio.EmergencyLiquidate()
			}

			portfolioStatus := globalTradingSystem.Portfolio.GetStatus()

			return map[string]interface{}{
				"success":          true,
				"message":          "🛑 Auto-trading stopped.",
				"loops_completed":  status.LoopCount,
				"total_decisions":  len(status.RecentDecisions),
				"positions_closed": len(closedTrades),
				"final_pnl":        portfolioStatus["total_pnl"],
				"portfolio_value":  portfolioStatus["total_value"],
			}, nil
		}).
		Build()
}

func createGetAutoTradingStatusTool() core.Tool {
	return tools.New("get_auto_trading_status").
		Description("Get the current status of autonomous trading, including running state, recent decisions, and analysis.").
		Schema(tools.ObjectSchema(map[string]interface{}{})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			if globalTradingSystem.AutoTrader == nil {
				return map[string]interface{}{
					"running": false,
					"message": "Auto-trading has not been started. Use start_auto_trading first.",
				}, nil
			}

			status := globalTradingSystem.AutoTrader.GetStatus()

			// Simplify for display
			result := map[string]interface{}{
				"running":          status.Running,
				"message":          status.Message,
				"loop_count":       status.LoopCount,
				"interval":         status.Interval,
				"assets_monitored": status.AssetsMonitored,
				"open_positions":   status.OpenPositions,
				"total_pnl":        fmt.Sprintf("$%.2f", status.TotalPnL),
			}

			if status.StartedAt != nil {
				result["started_at"] = status.StartedAt.Format(time.RFC3339)
			}
			if status.LastLoopAt != nil {
				result["last_loop_at"] = status.LastLoopAt.Format(time.RFC3339)
			}

			// Add recent decisions (simplified)
			if len(status.RecentDecisions) > 0 {
				decisions := make([]map[string]interface{}, 0)
				for _, d := range status.RecentDecisions {
					decisions = append(decisions, map[string]interface{}{
						"time":      d.Timestamp.Format("15:04:05"),
						"symbol":    d.Symbol,
						"action":    d.Action,
						"executed":  d.Executed,
						"reasoning": d.Reasoning,
					})
				}
				result["recent_decisions"] = decisions
			}

			// Add current analysis (simplified)
			if len(status.CurrentAnalysis) > 0 {
				analysis := make(map[string]interface{})
				for symbol, a := range status.CurrentAnalysis {
					analysis[symbol] = map[string]interface{}{
						"price":    fmt.Sprintf("$%.2f", a.Price),
						"rsi":      fmt.Sprintf("%.1f", a.RSI),
						"trend":    a.TrendSignal,
						"strength": a.Strength,
					}
				}
				result["current_analysis"] = analysis
			}

			return result, nil
		}).
		Build()
}

// ============================================================================
// WITHDRAW TRADING PROFITS TOOL
// ============================================================================

func createWithdrawTradingProfitsTool(liminalExecutor core.ToolExecutor) core.Tool {
	return tools.New("withdraw_trading_profits").
		Description("Withdraw funds from the trading account back to the Liminal wallet. Can withdraw specific amounts, profit only, or emergency close all positions.").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"amount":   tools.NumberProperty("Amount to withdraw in USD (optional - if omitted, defaults based on strategy)"),
			"strategy": tools.StringEnumProperty("Withdrawal strategy", "standard", "profits_only", "full_withdrawal"),
		}, "strategy")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Amount   float64 `json:"amount"`
				Strategy string  `json:"strategy"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			return executeWithdrawal(ctx, liminalExecutor, params.Amount, params.Strategy)
		}).
		Build()
}

// executeWithdrawal handles the core withdrawal logic
func executeWithdrawal(ctx context.Context, liminalExecutor core.ToolExecutor, amount float64, strategy string) (map[string]interface{}, error) {
	// Default strategy
	if strategy == "" {
		strategy = "standard"
	}

	userID := "user123" // TODO: Context

	// Create fund bridge
	fundBridge := trading.NewFundBridge(liminalExecutor, globalTradingSystem.Binance, globalTradingSystem.DB)
	portfolio := globalTradingSystem.Portfolio

	var withdrawAmount float64
	var message string

	portfolioStatus := portfolio.GetStatus()
	currentCash := portfolioStatus["available_cash"].(float64)
	totalPnL := portfolioStatus["total_pnl"].(float64)

	switch strategy {
	case "full_withdrawal":
		// Close all positions
		trades := portfolio.EmergencyLiquidate()
		log.Printf("🚨 Emergency liquidated %d positions for withdrawal", len(trades))

		// Re-fetch cash after liquidation
		portfolioStatus = portfolio.GetStatus()
		currentCash = portfolioStatus["available_cash"].(float64)
		withdrawAmount = currentCash
		message = fmt.Sprintf("✅ Full withdrawal initiated. Liquidated %d positions. Withdrawing $%.2f.", len(trades), withdrawAmount)

	case "profits_only":
		if totalPnL <= 0 {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("No profits to withdraw. Total P&L is $%.2f", totalPnL),
			}, nil
		}
		withdrawAmount = totalPnL
		if withdrawAmount > currentCash {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Profits ($%.2f) exceed available cash ($%.2f). You may need to close positions first.", withdrawAmount, currentCash),
			}, nil
		}
		message = fmt.Sprintf("✅ Profit withdrawal initiated: $%.2f", withdrawAmount)

	case "standard":
		if amount <= 0 {
			return map[string]interface{}{
				"success": false,
				"error":   "Amount must be greater than 0 for standard withdrawal",
			}, nil
		}
		withdrawAmount = amount
		if withdrawAmount > currentCash {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Insufficient available cash: have $%.2f, need $%.2f", currentCash, withdrawAmount),
			}, nil
		}
		message = fmt.Sprintf("✅ Withdrawal initiated: $%.2f", withdrawAmount)
	}

	if withdrawAmount <= 0 {
		return map[string]interface{}{
			"success": false,
			"error":   "Withdrawal amount must be greater than 0",
		}, nil
	}

	// Withdraw from portfolio
	if err := portfolio.Withdraw(withdrawAmount); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to withdraw from portfolio: %v", err),
		}, nil
	}

	// Transfer via bridge
	if err := fundBridge.TransferFromTrading(ctx, userID, withdrawAmount); err != nil {
		// Rollback portfolio withdrawal (simplified)
		//In production we'd need better transaction management
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to transfer funds: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":        true,
		"amount":         fmt.Sprintf("$%.2f", withdrawAmount),
		"message":        message,
		"remaining_cash": fmt.Sprintf("$%.2f", currentCash-withdrawAmount),
	}, nil
}
