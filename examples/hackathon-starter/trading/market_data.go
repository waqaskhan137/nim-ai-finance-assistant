// Package trading provides trading tools and subagents for autonomous trading.
package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
	
	// Import connectors for Binance integration
	"github.com/becomeliminal/nim-go-sdk/examples/hackathon-starter/trading/connectors"
)

// MockMarketData provides simulated market data for development/testing.
type MockMarketData struct {
	mu           sync.RWMutex
	prices       map[string]float64
	priceHistory map[string][]Candle
}

// Candle represents OHLCV data.
type Candle struct {
	Time   time.Time `json:"time"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Close  float64   `json:"close"`
	Volume float64   `json:"volume"`
}

// NewMockMarketData creates a new mock market data provider.
func NewMockMarketData() *MockMarketData {
	m := &MockMarketData{
		prices: map[string]float64{
			"BTCUSDT": 42500.00,
			"ETHUSDT": 2250.00,
			"XAUUSD":  2050.00,   // Gold
			"EURUSD":  1.0850,    // Forex
		},
		priceHistory: make(map[string][]Candle),
	}
	
	// Generate initial candle history
	for symbol, price := range m.prices {
		m.priceHistory[symbol] = generateMockCandles(price, 100)
	}
	
	return m
}

// generateMockCandles creates realistic-looking historical candles.
func generateMockCandles(currentPrice float64, count int) []Candle {
	candles := make([]Candle, count)
	price := currentPrice * 0.95 // Start slightly lower
	
	for i := 0; i < count; i++ {
		// Random walk with slight upward bias
		change := (rand.Float64() - 0.48) * (price * 0.02)
		open := price
		close := price + change
		
		high := math.Max(open, close) + rand.Float64()*(price*0.005)
		low := math.Min(open, close) - rand.Float64()*(price*0.005)
		volume := 1000 + rand.Float64()*10000
		
		candles[i] = Candle{
			Time:   time.Now().Add(-time.Duration(count-i) * time.Hour),
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}
		
		price = close
	}
	
	return candles
}

// GetPrice returns the current price for a symbol.
func (m *MockMarketData) GetPrice(symbol string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	price, ok := m.prices[symbol]
	if !ok {
		return 0, fmt.Errorf("unknown symbol: %s", symbol)
	}
	
	// Add small random fluctuation
	fluctuation := (rand.Float64() - 0.5) * (price * 0.001)
	return price + fluctuation, nil
}

// GetCandles returns historical candle data.
func (m *MockMarketData) GetCandles(symbol string, limit int) ([]Candle, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	candles, ok := m.priceHistory[symbol]
	if !ok {
		return nil, fmt.Errorf("unknown symbol: %s", symbol)
	}
	
	if limit > len(candles) {
		limit = len(candles)
	}
	
	// Return most recent candles
	return candles[len(candles)-limit:], nil
}

// SimulatePriceMove simulates a price movement (for testing stop-loss).
func (m *MockMarketData) SimulatePriceMove(symbol string, percentChange float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if price, ok := m.prices[symbol]; ok {
		m.prices[symbol] = price * (1 + percentChange/100)
	}
}

// ============================================================================
// MARKET DATA TOOLS
// ============================================================================

// CreateGetPriceTool creates a tool to get current price.
func CreateGetPriceTool(market MarketDataProvider) core.Tool {
	return tools.New("get_market_price").
		Description("Get the current price for a trading symbol (e.g., BTCUSDT, ETHUSDT, XAUUSD, EURUSD)").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol": tools.StringProperty("Trading symbol (e.g., BTCUSDT for Bitcoin, XAUUSD for Gold)"),
		}, "symbol")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}
			
			price, err := market.GetPrice(params.Symbol)
			if err != nil {
				return map[string]interface{}{
					"error": err.Error(),
				}, nil
			}
			
			return map[string]interface{}{
				"symbol": params.Symbol,
				"price":  fmt.Sprintf("%.4f", price),
				"time":   time.Now().Format(time.RFC3339),
			}, nil
		}).
		Build()
}

// CreateGetCandlesTool creates a tool to get historical candles.
func CreateGetCandlesTool(market MarketDataProvider) core.Tool {
	return tools.New("get_candles").
		Description("Get historical OHLCV candle data for technical analysis").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol": tools.StringProperty("Trading symbol (e.g., BTCUSDT)"),
			"limit":  tools.NumberProperty("Number of candles to retrieve (max 100)"),
		}, "symbol")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol string `json:"symbol"`
				Limit  int    `json:"limit"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}
			
			if params.Limit <= 0 {
				params.Limit = 20
			}
			if params.Limit > 100 {
				params.Limit = 100
			}
			
			candles, err := market.GetCandles(params.Symbol, params.Limit)
			if err != nil {
				return map[string]interface{}{
					"error": err.Error(),
				}, nil
			}
			
			// Simplify for AI consumption
			simplified := make([]map[string]interface{}, len(candles))
			for i, c := range candles {
				simplified[i] = map[string]interface{}{
					"time":   c.Time.Format("2006-01-02 15:04"),
					"open":   fmt.Sprintf("%.2f", c.Open),
					"high":   fmt.Sprintf("%.2f", c.High),
					"low":    fmt.Sprintf("%.2f", c.Low),
					"close":  fmt.Sprintf("%.2f", c.Close),
					"volume": fmt.Sprintf("%.0f", c.Volume),
				}
			}
			
			return map[string]interface{}{
				"symbol":  params.Symbol,
				"candles": simplified,
				"count":   len(simplified),
			}, nil
		}).
		Build()
}

// ============================================================================
// REAL MARKET DATA PROVIDER (Binance Integration)
// ============================================================================

// RealMarketData provides actual market data from Binance API.
type RealMarketData struct {
	mu      sync.RWMutex
	binance *connectors.BinanceConnector
	fallback *MockMarketData // Fallback to mock data if Binance fails
}

// NewRealMarketData creates a market data provider that uses real Binance data.
func NewRealMarketData(binance *connectors.BinanceConnector) *RealMarketData {
	return &RealMarketData{
		binance:  binance,
		fallback: NewMockMarketData(),
	}
}

// GetPrice returns the current price for a symbol (real Binance data).
func (r *RealMarketData) GetPrice(symbol string) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.binance != nil {
		price, err := r.binance.GetPrice(symbol)
		if err == nil {
			return price, nil
		}
		log.Printf("⚠️ Binance price fetch failed for %s: %v, using fallback", symbol, err)
	}

	// Fallback to mock data
	return r.fallback.GetPrice(symbol)
}

// GetCandles returns historical OHLCV candles (real Binance data).
func (r *RealMarketData) GetCandles(symbol string, count int) ([]Candle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.binance != nil {
		// Use 1h interval for analysis
		connectorCandles, err := r.binance.GetCandles(symbol, "1h", count)
		if err == nil && len(connectorCandles) > 0 {
			// Convert connectors.Candle to local Candle
			candles := make([]Candle, len(connectorCandles))
			for i, cc := range connectorCandles {
				candles[i] = Candle{
					Time:   time.Unix(cc.OpenTime/1000, 0), // Convert milliseconds to time.Time
					Open:   cc.Open,
					High:   cc.High,
					Low:    cc.Low,
					Close:  cc.Close,
					Volume: cc.Volume,
				}
			}
			return candles, nil
		}
		log.Printf("⚠️ Binance candles fetch failed for %s: %v, using fallback", symbol, err)
	}

	// Fallback to mock data
	return r.fallback.GetCandles(symbol, count)
}

// GetIndicators calculates technical indicators using real market data.
func (r *RealMarketData) GetIndicators(symbol string) (map[string]interface{}, error) {
	candles, err := r.GetCandles(symbol, 100)
	if err != nil {
		return nil, err
	}

	if len(candles) < 20 {
		return nil, fmt.Errorf("insufficient candle data for %s", symbol)
	}

	// Extract close prices for indicator calculations
	closes := make([]float64, len(candles))
	for i, c := range candles {
		closes[i] = c.Close
	}

	// Calculate indicators using real candle data
	rsi := CalculateRSI(closes, 14)
	macd, signal, histogram := CalculateMACD(closes)
	bb := CalculateBollingerBands(closes, 20, 2.0)
	sma20 := CalculateSMA(closes, 20)

	currentPrice := candles[len(candles)-1].Close

	return map[string]interface{}{
		"rsi":           rsi,
		"macd":          map[string]interface{}{
			"value":     macd,
			"signal":    signal,
			"histogram": histogram,
		},
		"bollinger_bands": bb,
		"sma20":         []float64{sma20},
		"current_price": currentPrice,
		"price_vs_sma20": currentPrice - sma20,
	}, nil
}

// ============================================================================
// UNIFIED MARKET DATA INTERFACE
// ============================================================================

// MarketDataProvider interface for both mock and real data.
type MarketDataProvider interface {
	GetPrice(symbol string) (float64, error)
	GetCandles(symbol string, count int) ([]Candle, error)
	GetIndicators(symbol string) (map[string]interface{}, error)
}

// MockMarketData implements MarketDataProvider.
func (m *MockMarketData) GetIndicators(symbol string) (map[string]interface{}, error) {
	return m.calculateMockIndicators(symbol)
}

// calculateMockIndicators provides mock technical indicators for testing.
func (m *MockMarketData) calculateMockIndicators(symbol string) (map[string]interface{}, error) {
	m.mu.RLock()
	price, exists := m.prices[symbol]
	m.mu.RUnlock()
	
	if !exists {
		price = 100.0 // Default fallback
	}

	// Generate mock indicators based on the symbol
	rsi := 50.0 + rand.Float64()*20 - 10 // 40-60 range
	macd := map[string]interface{}{
		"value":  rand.Float64()*10 - 5,
		"signal": rand.Float64()*10 - 5,
		"histogram": rand.Float64()*2 - 1,
	}
	
	bb := map[string]interface{}{
		"upper": price * 1.05,
		"middle": price,
		"lower": price * 0.95,
	}
	
	sma20 := price * (0.98 + rand.Float64()*0.04) // +/- 2% from price

	return map[string]interface{}{
		"rsi":           rsi,
		"macd":          macd,
		"bollinger_bands": bb,
		"sma20":         []float64{sma20},
		"current_price": price,
		"price_vs_sma20": price - sma20,
	}, nil
}
