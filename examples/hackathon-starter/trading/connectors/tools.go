package connectors

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/becomeliminal/nim-go-sdk/core"
	"github.com/becomeliminal/nim-go-sdk/tools"
)

// CreateBinanceTools creates all trading tools for a Binance connector.
func CreateBinanceTools(connector *BinanceConnector) []core.Tool {
	return []core.Tool{
		createBinancePriceTool(connector),
		createBinanceCandlesTool(connector),
		createBinanceBalanceTool(connector),
		createBinanceMarketOrderTool(connector),
		createBinanceOpenOrdersTool(connector),
	}
}

// createBinancePriceTool creates a tool to get live prices from Binance.
func createBinancePriceTool(connector *BinanceConnector) core.Tool {
	return tools.New("binance_price").
		Description("Get live price from Binance for a crypto symbol (e.g., BTCUSDT, ETHUSDT)").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol": tools.StringProperty("Trading pair symbol (e.g., BTCUSDT)"),
		}, "symbol")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol string `json:"symbol"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			price, err := connector.GetPrice(params.Symbol)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"success": true,
				"symbol":  params.Symbol,
				"price":   fmt.Sprintf("%.8f", price),
				"source":  connector.Name(),
				"testnet": connector.IsTestnet(),
			}, nil
		}).
		Build()
}

// createBinanceCandlesTool creates a tool to get historical candles.
func createBinanceCandlesTool(connector *BinanceConnector) core.Tool {
	return tools.New("binance_candles").
		Description("Get historical OHLCV candles from Binance").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol":   tools.StringProperty("Trading pair symbol (e.g., BTCUSDT)"),
			"interval": tools.StringProperty("Candle interval: 1m, 5m, 15m, 1h, 4h, 1d"),
			"limit":    tools.NumberProperty("Number of candles to fetch (max 1000)"),
		}, "symbol")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol   string `json:"symbol"`
				Interval string `json:"interval"`
				Limit    int    `json:"limit"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			if params.Interval == "" {
				params.Interval = "1h"
			}
			if params.Limit == 0 {
				params.Limit = 20
			}

			candles, err := connector.GetCandles(params.Symbol, params.Interval, params.Limit)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			// Convert to simpler format
			candleData := make([]map[string]interface{}, len(candles))
			for i, c := range candles {
				candleData[i] = map[string]interface{}{
					"open":   fmt.Sprintf("%.2f", c.Open),
					"high":   fmt.Sprintf("%.2f", c.High),
					"low":    fmt.Sprintf("%.2f", c.Low),
					"close":  fmt.Sprintf("%.2f", c.Close),
					"volume": fmt.Sprintf("%.2f", c.Volume),
				}
			}

			return map[string]interface{}{
				"success":  true,
				"symbol":   params.Symbol,
				"interval": params.Interval,
				"candles":  candleData,
				"count":    len(candles),
			}, nil
		}).
		Build()
}

// createBinanceBalanceTool creates a tool to get account balances.
func createBinanceBalanceTool(connector *BinanceConnector) core.Tool {
	return tools.New("binance_balance").
		Description("Get account balances from Binance (requires API key)").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"asset": tools.StringProperty("Optional: specific asset to check (e.g., BTC, USDT)"),
		})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Asset string `json:"asset"`
			}
			json.Unmarshal(input, &params) // Ignore error, asset is optional

			if params.Asset != "" {
				balance, err := connector.GetAssetBalance(params.Asset)
				if err != nil {
					return map[string]interface{}{
						"success": false,
						"error":   err.Error(),
					}, nil
				}
				return map[string]interface{}{
					"success": true,
					"asset":   balance.Asset,
					"free":    fmt.Sprintf("%.8f", balance.Free),
					"locked":  fmt.Sprintf("%.8f", balance.Locked),
					"testnet": connector.IsTestnet(),
				}, nil
			}

			balances, err := connector.GetBalances()
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			balanceData := make([]map[string]interface{}, len(balances))
			for i, b := range balances {
				balanceData[i] = map[string]interface{}{
					"asset":  b.Asset,
					"free":   fmt.Sprintf("%.8f", b.Free),
					"locked": fmt.Sprintf("%.8f", b.Locked),
				}
			}

			return map[string]interface{}{
				"success":  true,
				"balances": balanceData,
				"count":    len(balances),
				"testnet":  connector.IsTestnet(),
			}, nil
		}).
		Build()
}

// createBinanceMarketOrderTool creates a tool to place market orders.
func createBinanceMarketOrderTool(connector *BinanceConnector) core.Tool {
	return tools.New("binance_market_order").
		Description("Place a market order on Binance. Requires user confirmation.").
		RequiresConfirmation().
		SummaryTemplate("Authorize Market Order").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol":   tools.StringProperty("Trading pair (e.g., BTCUSDT)"),
			"side":     tools.StringProperty("BUY or SELL"),
			"quantity": tools.NumberProperty("Quantity to trade"),
		}, "symbol", "side", "quantity")).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol   string  `json:"symbol"`
				Side     string  `json:"side"`
				Quantity float64 `json:"quantity"`
			}
			if err := json.Unmarshal(input, &params); err != nil {
				return nil, err
			}

			order, err := connector.PlaceMarketOrder(params.Symbol, params.Side, params.Quantity)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"success":      true,
				"order_id":     order.ID,
				"symbol":       order.Symbol,
				"side":         order.Side,
				"status":       order.Status,
				"executed_qty": fmt.Sprintf("%.8f", order.ExecutedQty),
				"testnet":      connector.IsTestnet(),
			}, nil
		}).
		Build()
}

// createBinanceOpenOrdersTool creates a tool to get open orders.
func createBinanceOpenOrdersTool(connector *BinanceConnector) core.Tool {
	return tools.New("binance_open_orders").
		Description("Get all open orders on Binance").
		Schema(tools.ObjectSchema(map[string]interface{}{
			"symbol": tools.StringProperty("Optional: filter by symbol"),
		})).
		HandlerFunc(func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var params struct {
				Symbol string `json:"symbol"`
			}
			json.Unmarshal(input, &params)

			orders, err := connector.GetOpenOrders(params.Symbol)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}, nil
			}

			orderData := make([]map[string]interface{}, len(orders))
			for i, o := range orders {
				orderData[i] = map[string]interface{}{
					"order_id": o.ID,
					"symbol":   o.Symbol,
					"side":     o.Side,
					"type":     o.Type,
					"quantity": fmt.Sprintf("%.8f", o.Quantity),
					"price":    fmt.Sprintf("%.8f", o.Price),
					"status":   o.Status,
				}
			}

			return map[string]interface{}{
				"success": true,
				"orders":  orderData,
				"count":   len(orders),
				"testnet": connector.IsTestnet(),
			}, nil
		}).
		Build()
}
