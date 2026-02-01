# Nim - AI Financial Assistant

> **Hack SouthWest 2026 Submission** - Built for the Liminal Vibe Banking Hackathon

Nim is an AI-powered financial assistant that guides users through a proven financial journey: **Stabilize → Save → Invest**. Built on Anthropic's Claude AI with Liminal's stablecoin banking platform.

## The Philosophy

Most fintech apps push users toward trading immediately. Nim takes a different approach:

1. **Stabilize** - Understand your cash flow, create a budget, eliminate wasteful subscriptions
2. **Save** - Build an emergency fund through automated savings rules
3. **Invest** - Only after completing Stage 1 emergency fund (2 weeks of expenses)

This ensures users build a solid financial foundation before taking on investment risk.

---

## Features

### Chat with Nim
- Conversational AI assistant powered by Claude
- Natural language financial guidance
- Floating chat widget accessible from any page
- Full chat page with conversation history

### Budget Planner
- Envelope-based budgeting (Needs/Wants/Bills/Goals)
- Configurable guardrails (soft alerts, hard stops)
- Synced between chat and UI
- Visual progress tracking

### Smart Savings
- **Automated savings rules:**
  - Payday Sweep - Move fixed amount after income
  - Round-ups - Save spare change from purchases
  - Weekly/Monthly scheduled transfers
- **Emergency Fund Ladder:**
  - Stage 1: 2 weeks expenses (unlocks investing)
  - Stage 2: 1 month expenses
  - Stage 3: 3 months expenses (fully funded)

### Subscription Management
- Auto-detect recurring subscriptions
- Identify forgotten/unused services
- Track price increases
- Calculate potential savings

### Trading Terminal (Gated)
- Only accessible after Stage 1 emergency fund
- Real-time market data (BTC, ETH, Gold, EUR)
- Technical indicators (RSI, MACD, Bollinger Bands)
- Paper trading with Binance Testnet
- Risk-managed position sizing

---

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- Anthropic API key
- Liminal account (via TestFlight app)

### 1. Clone the repository
```bash
git clone https://github.com/becomeliminal/nim-go-sdk.git
cd nim-go-sdk/examples/hackathon-starter
```

### 2. Set up environment
```bash
cp .env.example .env
```

Edit `.env` and add your API keys:
```env
ANTHROPIC_API_KEY=sk-ant-...

# Optional: Binance Testnet for paper trading
BINANCE_API_KEY=your_testnet_key
BINANCE_API_SECRET=your_testnet_secret
BINANCE_TESTNET=true
```

### 3. Run the backend
```bash
go build -o server . && ./server
```

You should see:
```
✅ Journey schema initialized (user plans, budgets, savings rules, emergency funds)
✅ Added 11 journey tools (onboarding, budget, savings rules, emergency fund, investment surplus, autopilot)
🚀 Trading-Enabled Hackathon Starter Running
📡 WebSocket endpoint: ws://localhost:8081/ws
```

### 4. Run the frontend
```bash
cd frontend
npm install
npm run dev
```

### 5. Open in browser
Navigate to **http://localhost:5173**

---

## Project Structure

```
nim-go-sdk/
├── core/                    # Core interfaces (Agent, Tool, Context)
├── engine/                  # Claude API integration, agentic loop
├── tools/                   # Tool builder, schema helpers, Liminal tools
├── server/                  # WebSocket server, streaming protocol
├── subagent/                # Sub-agent framework
│
└── examples/hackathon-starter/
    ├── main.go              # Server entry point
    ├── trading/
    │   ├── journey.go       # Journey data models & DB schema
    │   ├── journey_tools.go # AI tools for journey flow
    │   ├── portfolio.go     # Trading portfolio management
    │   └── ...
    │
    └── frontend/
        ├── main.tsx         # React app entry
        ├── ChatWidget.tsx   # Floating chat bubble
        ├── ChatPage.tsx     # Full chat interface
        ├── BudgetPage.tsx   # Budget planner UI
        ├── SmartSavingsPage.tsx  # Savings rules & emergency fund
        └── ...
```

---

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `ws://localhost:8081/ws` | WebSocket for chat |
| `/api/journey-status` | User's progress through the journey |
| `/api/budget` | GET/POST budget from journey DB |
| `/api/savings-rules` | GET/POST automated savings rules |
| `/api/emergency-fund` | Emergency fund ladder status |
| `/api/budget-plan` | AI-generated budget analysis |
| `/api/savings-analysis` | Savings optimization analysis |
| `/api/subscriptions` | Detected subscriptions |
| `/api/trading-status` | Trading portfolio status |

---

## AI Tools (43 Total)

### Journey Tools (11)
- `create_user_plan` / `get_user_plan` - Onboarding
- `get_journey_status` - Progress tracking
- `create_budget` / `get_budget_status` - Budget management
- `create_savings_rule` / `get_savings_rules` - Automated savings
- `get_emergency_fund_status` - Emergency fund ladder
- `get_investment_surplus` - Safe amount to invest
- `get_weekly_digest` / `approve_pending_action` - Autopilot

### Liminal Banking Tools (9)
- `get_balance`, `get_savings_balance`, `get_vault_rates`
- `get_transactions`, `get_profile`, `search_users`
- `send_money`, `deposit_savings`, `withdraw_savings`

### Insight Tools (9)
- `analyze_spending_patterns` - Spending trends & anomalies
- `optimize_savings` - Find idle cash, compare rates
- `calculate_financial_health` - Comprehensive health score
- `assess_trading_readiness` - Should you trade?
- `get_smart_budget` - AI budget recommendations
- `create_budget_plan` - Wealth forecasting
- `detect_subscriptions` - Find recurring charges
- `get_savings_analysis` / `execute_savings_sweep`

### Trading Tools (14)
- Market data, indicators, portfolio management
- Autonomous trading with risk controls
- Binance integration (testnet)

---

## Tech Stack

- **Backend:** Go 1.21
- **AI:** Anthropic Claude (claude-sonnet-4-20250514)
- **Frontend:** React 18, TypeScript, Vite
- **Database:** SQLite (conversation history, journey data)
- **Banking API:** Liminal
- **Exchange:** Binance Testnet (paper trading)
- **Design System:** Apple HIG-inspired, dark/light mode

---

## What We Built for the Hackathon

### The Problem
Most financial apps either:
- Focus only on budgeting (boring, no growth)
- Push users into trading too early (risky)
- Don't connect the dots between saving and investing

### Our Solution
Nim guides users through a complete financial journey:

1. **Onboarding** - Conversational goal-setting with Nim
2. **Budget** - Envelope-based system synced with chat
3. **Subscriptions** - Find and eliminate waste
4. **Smart Savings** - Automated rules to build emergency fund
5. **Trading Terminal** - Only after emergency fund Stage 1
6. **Autopilot** - Weekly digests and automated actions

### Key Innovations
- **Gated investing** - Must complete Stage 1 emergency fund first
- **Chat ↔ UI sync** - Changes in chat reflect in UI and vice versa
- **Floating chat widget** - Access Nim from any page
- **Emergency fund ladder** - Visual 3-stage progress tracking
- **Guardrails, not guilt** - Soft alerts and hard stops on spending

---

## Demo

1. Open the app and login with your Liminal account
2. Click the chat bubble and say "Help me get started"
3. Nim will guide you through onboarding
4. Create a budget via chat or the Budget page
5. Set up savings rules on the Smart Savings page
6. Watch your emergency fund grow
7. Once Stage 1 is complete, try asking about trading!

---

## Team

Built with caffeine and Claude at Hack SouthWest 2026.

---

## License

MIT
