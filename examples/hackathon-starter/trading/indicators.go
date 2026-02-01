package trading

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// ============================================================================
// TECHNICAL INDICATORS
// ============================================================================

// CalculateSMA computes Simple Moving Average.
func CalculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// CalculateRSI computes Relative Strength Index.
func CalculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50 // Neutral if not enough data
	}
	
	gains := 0.0
	losses := 0.0
	
	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	
	if losses == 0 {
		return 100
	}
	
	rs := (gains / float64(period)) / (losses / float64(period))
	return 100 - (100 / (1 + rs))
}

// CalculateMACD computes MACD indicator.
func CalculateMACD(prices []float64) (macd, signal, histogram float64) {
	if len(prices) < 26 {
		return 0, 0, 0
	}
	
	ema12 := calculateEMA(prices, 12)
	ema26 := calculateEMA(prices, 26)
	macd = ema12 - ema26
	
	// For simplicity, signal is SMA of last 9 MACD values
	// In production, you'd track historical MACD values
	signal = macd * 0.9 // Simplified
	histogram = macd - signal
	
	return macd, signal, histogram
}

// calculateEMA computes Exponential Moving Average.
func calculateEMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}
	
	multiplier := 2.0 / float64(period+1)
	ema := CalculateSMA(prices[:period], period)
	
	for i := period; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}
	
	return ema
}

// BollingerBands holds Bollinger Band values.
type BollingerBands struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
}

// CalculateBollingerBands computes Bollinger Bands.
func CalculateBollingerBands(prices []float64, period int, stdDev float64) BollingerBands {
	if len(prices) < period {
		return BollingerBands{}
	}
	
	sma := CalculateSMA(prices, period)
	
	// Calculate standard deviation
	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += math.Pow(prices[i]-sma, 2)
	}
	std := math.Sqrt(sum / float64(period))
	
	return BollingerBands{
		Upper:  sma + (std * stdDev),
		Middle: sma,
		Lower:  sma - (std * stdDev),
	}
}

// ============================================================================
// INDICATOR TOOLS
// ============================================================================

// CreateCalcIndicatorsTool creates a tool to calculate multiple indicators.
func CreateCalcIndicatorsTool(market MarketDataProvider) core.Tool {
	return tools.New("calc_indicators").
		Description("Calculate technical indicators (RSI, SMA, MACD, Bollinger Bands) for a symbol").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol": tools.StringProperty("Trading symbol (e.g., BTCUSDT)"),
		}, "symbol")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}
			
			candles, err := market.GetCandles(params.Symbol, 50)
			if err != nil {
				return map[string]interface{}{
					"error": err.Error(),
				}, nil
			}
			
			// Extract close prices
			closes := make([]float64, len(candles))
			for i, c := range candles {
				closes[i] = c.Close
			}
			
			currentPrice := closes[len(closes)-1]
			rsi := CalculateRSI(closes, 14)
			sma20 := CalculateSMA(closes, 20)
			macd, signal, histogram := CalculateMACD(closes)
			bb := CalculateBollingerBands(closes, 20, 2)
			
			// Determine signals
			rsiSignal := "neutral"
			if rsi < 30 {
				rsiSignal = "oversold (potential buy)"
			} else if rsi > 70 {
				rsiSignal = "overbought (potential sell)"
			}
			
			trendSignal := "neutral"
			if currentPrice > sma20 {
				trendSignal = "bullish (price above SMA20)"
			} else {
				trendSignal = "bearish (price below SMA20)"
			}
			
			macdSignal := "neutral"
			if histogram > 0 {
				macdSignal = "bullish momentum"
			} else {
				macdSignal = "bearish momentum"
			}
			
			return map[string]interface{}{
				"symbol":        params.Symbol,
				"current_price": fmt.Sprintf("%.2f", currentPrice),
				"indicators": map[string]interface{}{
					"rsi": map[string]interface{}{
						"value":  fmt.Sprintf("%.2f", rsi),
						"signal": rsiSignal,
					},
					"sma20": map[string]interface{}{
						"value":  fmt.Sprintf("%.2f", sma20),
						"signal": trendSignal,
					},
					"macd": map[string]interface{}{
						"macd":      fmt.Sprintf("%.4f", macd),
						"signal":    fmt.Sprintf("%.4f", signal),
						"histogram": fmt.Sprintf("%.4f", histogram),
						"trend":     macdSignal,
					},
					"bollinger": map[string]interface{}{
						"upper":  fmt.Sprintf("%.2f", bb.Upper),
						"middle": fmt.Sprintf("%.2f", bb.Middle),
						"lower":  fmt.Sprintf("%.2f", bb.Lower),
					},
				},
			}, nil
		}).
		Build()
}
