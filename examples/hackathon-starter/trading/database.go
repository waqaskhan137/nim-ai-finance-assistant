package trading

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database handles SQLite persistence for trading data.
type Database struct {
	db *sql.DB
}

// NewDatabase creates and initializes the SQLite database.
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &Database{db: db}
	if err := d.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return d, nil
}

// initialize creates the database schema.
func (d *Database) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS trades (
		id TEXT PRIMARY KEY,
		symbol TEXT NOT NULL,
		side TEXT NOT NULL,
		entry_price REAL NOT NULL,
		exit_price REAL,
		quantity REAL NOT NULL,
		pnl REAL DEFAULT 0,
		open_time DATETIME NOT NULL,
		close_time DATETIME,
		status TEXT DEFAULT 'open',
		stop_loss REAL,
		take_profit REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS portfolio_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		total_value REAL NOT NULL,
		cash REAL NOT NULL,
		total_pnl REAL NOT NULL,
		open_positions INTEGER NOT NULL,
		risk_profile TEXT NOT NULL,
		snapshot_time DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trading_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		initial_budget REAL NOT NULL,
		stop_loss_floor REAL NOT NULL,
		risk_profile TEXT NOT NULL,
		final_value REAL,
		total_pnl REAL DEFAULT 0,
		trades_count INTEGER DEFAULT 0,
		win_count INTEGER DEFAULT 0,
		loss_count INTEGER DEFAULT 0,
		start_time DATETIME DEFAULT CURRENT_TIMESTAMP,
		end_time DATETIME,
		status TEXT DEFAULT 'active'
	);

	CREATE TABLE IF NOT EXISTS trading_allocations (
		user_id TEXT PRIMARY KEY,
		amount REAL NOT NULL,
		stop_loss_floor REAL NOT NULL,
		risk_profile TEXT NOT NULL,
		binance_account TEXT,
		status TEXT DEFAULT 'active',
		allocated_at DATETIME NOT NULL,
		closed_at DATETIME,
		final_value REAL,
		total_pnl REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trading_preferences (
		user_id TEXT PRIMARY KEY,
		assets TEXT NOT NULL, -- JSON array of asset symbols
		style TEXT NOT NULL,
		risk_profile TEXT NOT NULL,
		profit_target REAL NOT NULL,
		max_loss_percent REAL NOT NULL,
		auto_trade BOOLEAN DEFAULT 0,
		trading_hours TEXT, -- JSON TimeRange object
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
	CREATE INDEX IF NOT EXISTS idx_trades_status ON trades(status);
	CREATE INDEX IF NOT EXISTS idx_trades_open_time ON trades(open_time);
	CREATE INDEX IF NOT EXISTS idx_snapshots_time ON portfolio_snapshots(snapshot_time);
	CREATE INDEX IF NOT EXISTS idx_allocations_user ON trading_allocations(user_id);
	CREATE INDEX IF NOT EXISTS idx_preferences_user ON trading_preferences(user_id);
	`

	_, err := d.db.Exec(schema)
	return err
}

// SaveTrade saves a trade to the database.
func (d *Database) SaveTrade(trade *Trade, status string) error {
	query := `
		INSERT OR REPLACE INTO trades 
		(id, symbol, side, entry_price, exit_price, quantity, pnl, open_time, close_time, status, stop_loss, take_profit)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var exitPrice, pnl sql.NullFloat64
	var closeTime sql.NullTime
	var stopLoss, takeProfit sql.NullFloat64

	if trade.ExitPrice > 0 {
		exitPrice = sql.NullFloat64{Float64: trade.ExitPrice, Valid: true}
		pnl = sql.NullFloat64{Float64: trade.PnL, Valid: true}
		closeTime = sql.NullTime{Time: trade.CloseTime, Valid: true}
	}

	_, err := d.db.Exec(query,
		trade.ID,
		trade.Symbol,
		trade.Side,
		trade.EntryPrice,
		exitPrice,
		trade.Quantity,
		pnl,
		trade.OpenTime,
		closeTime,
		status,
		stopLoss,
		takeProfit,
	)

	return err
}

// SavePosition saves an open position as a trade record.
func (d *Database) SavePosition(pos *Position) error {
	query := `
		INSERT OR REPLACE INTO trades 
		(id, symbol, side, entry_price, quantity, open_time, status, stop_loss, take_profit)
		VALUES (?, ?, ?, ?, ?, ?, 'open', ?, ?)
	`

	_, err := d.db.Exec(query,
		pos.ID,
		pos.Symbol,
		pos.Side,
		pos.EntryPrice,
		pos.Quantity,
		pos.OpenTime,
		pos.StopLoss,
		pos.TakeProfit,
	)

	return err
}

// CloseTradeInDB updates a trade as closed.
func (d *Database) CloseTradeInDB(trade *Trade) error {
	query := `
		UPDATE trades 
		SET exit_price = ?, pnl = ?, close_time = ?, status = 'closed'
		WHERE id = ?
	`

	_, err := d.db.Exec(query,
		trade.ExitPrice,
		trade.PnL,
		trade.CloseTime,
		trade.ID,
	)

	return err
}

// SavePortfolioSnapshot saves current portfolio state.
func (d *Database) SavePortfolioSnapshot(totalValue, cash, pnl float64, openPositions int, riskProfile string) error {
	query := `
		INSERT INTO portfolio_snapshots 
		(total_value, cash, total_pnl, open_positions, risk_profile)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(query, totalValue, cash, pnl, openPositions, riskProfile)
	return err
}

// StartSession creates a new trading session.
func (d *Database) StartSession(budget, floor float64, profile string) (int64, error) {
	query := `
		INSERT INTO trading_sessions 
		(initial_budget, stop_loss_floor, risk_profile)
		VALUES (?, ?, ?)
	`

	result, err := d.db.Exec(query, budget, floor, profile)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// EndSession closes a trading session.
func (d *Database) EndSession(sessionID int64, finalValue, pnl float64, trades, wins, losses int) error {
	query := `
		UPDATE trading_sessions 
		SET final_value = ?, total_pnl = ?, trades_count = ?, 
		    win_count = ?, loss_count = ?, end_time = ?, status = 'closed'
		WHERE id = ?
	`

	_, err := d.db.Exec(query, finalValue, pnl, trades, wins, losses, time.Now(), sessionID)
	return err
}

// GetTradeHistory returns all closed trades.
func (d *Database) GetTradeHistory(limit int) ([]Trade, error) {
	query := `
		SELECT id, symbol, side, entry_price, exit_price, quantity, pnl, open_time, close_time
		FROM trades 
		WHERE status = 'closed'
		ORDER BY close_time DESC
		LIMIT ?
	`

	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []Trade
	for rows.Next() {
		var t Trade
		var exitPrice, pnl sql.NullFloat64
		var closeTime sql.NullTime

		err := rows.Scan(&t.ID, &t.Symbol, &t.Side, &t.EntryPrice, &exitPrice, &t.Quantity, &pnl, &t.OpenTime, &closeTime)
		if err != nil {
			continue
		}

		if exitPrice.Valid {
			t.ExitPrice = exitPrice.Float64
		}
		if pnl.Valid {
			t.PnL = pnl.Float64
		}
		if closeTime.Valid {
			t.CloseTime = closeTime.Time
		}

		trades = append(trades, t)
	}

	return trades, nil
}

// GetPnLSummary returns P&L statistics.
func (d *Database) GetPnLSummary() (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_trades,
			COALESCE(SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END), 0) as wins,
			COALESCE(SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END), 0) as losses,
			COALESCE(SUM(pnl), 0) as total_pnl,
			COALESCE(AVG(pnl), 0) as avg_pnl,
			COALESCE(MAX(pnl), 0) as best_trade,
			COALESCE(MIN(pnl), 0) as worst_trade
		FROM trades 
		WHERE status = 'closed'
	`

	var totalTrades, wins, losses int
	var totalPnL, avgPnL, bestTrade, worstTrade float64

	err := d.db.QueryRow(query).Scan(&totalTrades, &wins, &losses, &totalPnL, &avgPnL, &bestTrade, &worstTrade)
	if err != nil {
		return nil, err
	}

	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(wins) / float64(totalTrades) * 100
	}

	return map[string]interface{}{
		"total_trades": totalTrades,
		"wins":         wins,
		"losses":       losses,
		"total_pnl":    fmt.Sprintf("%.2f", totalPnL),
		"avg_pnl":      fmt.Sprintf("%.2f", avgPnL),
		"best_trade":   fmt.Sprintf("%.2f", bestTrade),
		"worst_trade":  fmt.Sprintf("%.2f", worstTrade),
		"win_rate":     fmt.Sprintf("%.1f%%", winRate),
	}, nil
}

// GetOpenPositions returns all open positions from DB.
func (d *Database) GetOpenPositions() ([]Position, error) {
	query := `
		SELECT id, symbol, side, entry_price, quantity, open_time, stop_loss, take_profit
		FROM trades 
		WHERE status = 'open'
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []Position
	for rows.Next() {
		var p Position
		var stopLoss, takeProfit sql.NullFloat64

		err := rows.Scan(&p.ID, &p.Symbol, &p.Side, &p.EntryPrice, &p.Quantity, &p.OpenTime, &stopLoss, &takeProfit)
		if err != nil {
			continue
		}

		if stopLoss.Valid {
			p.StopLoss = stopLoss.Float64
		}
		if takeProfit.Valid {
			p.TakeProfit = takeProfit.Float64
		}

		positions = append(positions, p)
	}

	return positions, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	return d.db.Close()
}

// ============================================================================
// TRADING ALLOCATIONS
// ============================================================================

// SaveAllocation saves a trading allocation to the database.
func (d *Database) SaveAllocation(allocation *TradingAllocation) error {
	query := `
		INSERT OR REPLACE INTO trading_allocations
		(user_id, amount, stop_loss_floor, risk_profile, binance_account, status, allocated_at, closed_at, final_value, total_pnl)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var closedAt, finalValue, totalPnL interface{}
	if allocation.ClosedAt != nil {
		closedAt = *allocation.ClosedAt
	} else {
		closedAt = nil
	}
	if allocation.FinalValue != nil {
		finalValue = *allocation.FinalValue
	} else {
		finalValue = nil
	}
	if allocation.TotalPnL != nil {
		totalPnL = *allocation.TotalPnL
	} else {
		totalPnL = nil
	}

	_, err := d.db.Exec(query,
		allocation.UserID,
		allocation.Amount,
		allocation.StopLossFloor,
		allocation.RiskProfile,
		allocation.BinanceAccount,
		allocation.Status,
		allocation.AllocatedAt,
		closedAt,
		finalValue,
		totalPnL,
	)
	return err
}

// GetAllocation retrieves a trading allocation for a user.
func (d *Database) GetAllocation(userID string) (*TradingAllocation, error) {
	query := `
		SELECT user_id, amount, stop_loss_floor, risk_profile, binance_account, status, allocated_at, closed_at, final_value, total_pnl
		FROM trading_allocations
		WHERE user_id = ?
	`

	var allocation TradingAllocation
	var closedAt sql.NullTime
	var finalVal, totalPnl sql.NullFloat64

	row := d.db.QueryRow(query, userID)
	err := row.Scan(
		&allocation.UserID,
		&allocation.Amount,
		&allocation.StopLossFloor,
		&allocation.RiskProfile,
		&allocation.BinanceAccount,
		&allocation.Status,
		&allocation.AllocatedAt,
		&closedAt,
		&finalVal,
		&totalPnl,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No allocation found
		}
		return nil, err
	}

	if closedAt.Valid {
		allocation.ClosedAt = &closedAt.Time
	}
	if finalVal.Valid {
		allocation.FinalValue = &finalVal.Float64
	}
	if totalPnl.Valid {
		allocation.TotalPnL = &totalPnl.Float64
	}

	return &allocation, nil
}

// CloseAllocation marks an allocation as closed with final results.
func (d *Database) CloseAllocation(userID string, finalValue float64) error {
	now := time.Now()
	totalPnL := finalValue - 0 // This would need to be calculated properly

	query := `
		UPDATE trading_allocations
		SET status = 'closed', closed_at = ?, final_value = ?, total_pnl = ?
		WHERE user_id = ?
	`

	_, err := d.db.Exec(query, now, finalValue, totalPnL, userID)
	return err
}

// ============================================================================
// TRADING PREFERENCES
// ============================================================================

// SavePreferences saves trading preferences to the database.
func (d *Database) SavePreferences(prefs *TradingPreferences) error {
	assetsJSON, err := json.Marshal(prefs.Assets)
	if err != nil {
		return err
	}

	var tradingHoursJSON []byte
	if prefs.TradingHours != nil {
		tradingHoursJSON, err = json.Marshal(prefs.TradingHours)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT OR REPLACE INTO trading_preferences
		(user_id, assets, style, risk_profile, profit_target, max_loss_percent, auto_trade, trading_hours, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.Exec(query,
		prefs.UserID,
		string(assetsJSON),
		prefs.Style,
		prefs.RiskProfile,
		prefs.ProfitTarget,
		prefs.MaxLossPercent,
		prefs.AutoTrade,
		string(tradingHoursJSON),
		prefs.UpdatedAt,
	)
	return err
}

// GetPreferences retrieves trading preferences for a user.
func (d *Database) GetPreferences(userID string) (*TradingPreferences, error) {
	query := `
		SELECT user_id, assets, style, risk_profile, profit_target, max_loss_percent, auto_trade, trading_hours, created_at, updated_at
		FROM trading_preferences
		WHERE user_id = ?
	`

	var prefs TradingPreferences
	var assetsJSON, tradingHoursJSON string

	row := d.db.QueryRow(query, userID)
	err := row.Scan(
		&prefs.UserID,
		&assetsJSON,
		&prefs.Style,
		&prefs.RiskProfile,
		&prefs.ProfitTarget,
		&prefs.MaxLossPercent,
		&prefs.AutoTrade,
		&tradingHoursJSON,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No preferences found
		}
		return nil, err
	}

	// Parse JSON fields
	if err := json.Unmarshal([]byte(assetsJSON), &prefs.Assets); err != nil {
		return nil, err
	}

	if tradingHoursJSON != "" {
		prefs.TradingHours = &TimeRange{}
		if err := json.Unmarshal([]byte(tradingHoursJSON), prefs.TradingHours); err != nil {
			return nil, err
		}
	}

	return &prefs, nil
}

// LogTrade is a helper to log trade actions.
func LogTrade(action string, trade interface{}) {
	log.Printf("[TRADE] %s: %+v", action, trade)
}
