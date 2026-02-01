package connectors

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// BinanceConfig holds Binance API configuration.
type BinanceConfig struct {
	APIKey    string
	APISecret string
	Testnet   bool // Use testnet endpoints
}

// BinanceConnector implements ExchangeConnector for Binance.
type BinanceConnector struct {
	config     BinanceConfig
	baseURL    string
	httpClient *http.Client
}

// Binance API endpoints
const (
	BinanceMainnetURL = "https://api.binance.com"
	BinanceTestnetURL = "https://testnet.binance.vision"
)

// NewBinanceConnector creates a new Binance connector.
func NewBinanceConnector(config BinanceConfig) *BinanceConnector {
	baseURL := BinanceMainnetURL
	if config.Testnet {
		baseURL = BinanceTestnetURL
	}

	return &BinanceConnector{
		config:  config,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ============================================================================
// INTERFACE IMPLEMENTATION
// ============================================================================

func (b *BinanceConnector) Name() string {
	if b.config.Testnet {
		return "Binance Testnet"
	}
	return "Binance"
}

func (b *BinanceConnector) SupportedSymbols() []string {
	return []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "XRPUSDT", "ADAUSDT",
		"DOGEUSDT", "SOLUSDT", "DOTUSDT", "MATICUSDT", "LTCUSDT",
	}
}

func (b *BinanceConnector) IsTestnet() bool {
	return b.config.Testnet
}

// ============================================================================
// MARKET DATA (Public endpoints - no auth required)
// ============================================================================

// GetPrice returns the current price for a symbol.
func (b *BinanceConnector) GetPrice(symbol string) (float64, error) {
	endpoint := "/api/v3/ticker/price"
	params := url.Values{}
	params.Set("symbol", symbol)

	var result struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	}

	if err := b.publicRequest("GET", endpoint, params, &result); err != nil {
		return 0, err
	}

	return strconv.ParseFloat(result.Price, 64)
}

// GetCandles returns historical OHLCV candles.
func (b *BinanceConnector) GetCandles(symbol string, interval string, limit int) ([]Candle, error) {
	endpoint := "/api/v3/klines"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval) // 1m, 5m, 15m, 1h, 4h, 1d
	params.Set("limit", strconv.Itoa(limit))

	var rawCandles [][]interface{}
	if err := b.publicRequest("GET", endpoint, params, &rawCandles); err != nil {
		return nil, err
	}

	candles := make([]Candle, len(rawCandles))
	for i, raw := range rawCandles {
		candles[i] = Candle{
			OpenTime:  int64(raw[0].(float64)),
			Open:      parseFloat(raw[1]),
			High:      parseFloat(raw[2]),
			Low:       parseFloat(raw[3]),
			Close:     parseFloat(raw[4]),
			Volume:    parseFloat(raw[5]),
			CloseTime: int64(raw[6].(float64)),
		}
	}

	return candles, nil
}

// GetBestBidAsk returns the best bid and ask prices.
func (b *BinanceConnector) GetBestBidAsk(symbol string) (bid, ask float64, err error) {
	endpoint := "/api/v3/ticker/bookTicker"
	params := url.Values{}
	params.Set("symbol", symbol)

	var result struct {
		BidPrice string `json:"bidPrice"`
		AskPrice string `json:"askPrice"`
	}

	if err := b.publicRequest("GET", endpoint, params, &result); err != nil {
		return 0, 0, err
	}

	bid, _ = strconv.ParseFloat(result.BidPrice, 64)
	ask, _ = strconv.ParseFloat(result.AskPrice, 64)
	return bid, ask, nil
}

// ============================================================================
// TRADING (Authenticated endpoints)
// ============================================================================

// PlaceMarketOrder places a market order.
func (b *BinanceConnector) PlaceMarketOrder(symbol, side string, quantity float64) (*Order, error) {
	endpoint := "/api/v3/order"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side) // BUY or SELL
	params.Set("type", "MARKET")
	params.Set("quantity", fmt.Sprintf("%.8f", quantity))

	var result binanceOrderResponse
	if err := b.signedRequest("POST", endpoint, params, &result); err != nil {
		return nil, err
	}

	return result.toOrder(), nil
}

// PlaceLimitOrder places a limit order.
func (b *BinanceConnector) PlaceLimitOrder(symbol, side string, quantity, price float64) (*Order, error) {
	endpoint := "/api/v3/order"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", "LIMIT")
	params.Set("timeInForce", "GTC") // Good til cancelled
	params.Set("quantity", fmt.Sprintf("%.8f", quantity))
	params.Set("price", fmt.Sprintf("%.8f", price))

	var result binanceOrderResponse
	if err := b.signedRequest("POST", endpoint, params, &result); err != nil {
		return nil, err
	}

	return result.toOrder(), nil
}

// CancelOrder cancels an open order.
func (b *BinanceConnector) CancelOrder(symbol, orderID string) error {
	endpoint := "/api/v3/order"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderID)

	var result map[string]interface{}
	return b.signedRequest("DELETE", endpoint, params, &result)
}

// GetOrder gets order status.
func (b *BinanceConnector) GetOrder(symbol, orderID string) (*Order, error) {
	endpoint := "/api/v3/order"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", orderID)

	var result binanceOrderResponse
	if err := b.signedRequest("GET", endpoint, params, &result); err != nil {
		return nil, err
	}

	return result.toOrder(), nil
}

// GetOpenOrders returns all open orders.
func (b *BinanceConnector) GetOpenOrders(symbol string) ([]Order, error) {
	endpoint := "/api/v3/openOrders"
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}

	var results []binanceOrderResponse
	if err := b.signedRequest("GET", endpoint, params, &results); err != nil {
		return nil, err
	}

	orders := make([]Order, len(results))
	for i, r := range results {
		orders[i] = *r.toOrder()
	}
	return orders, nil
}

// ============================================================================
// ACCOUNT
// ============================================================================

// GetBalances returns all account balances.
func (b *BinanceConnector) GetBalances() ([]Balance, error) {
	endpoint := "/api/v3/account"
	params := url.Values{}

	var result struct {
		Balances []struct {
			Asset  string `json:"asset"`
			Free   string `json:"free"`
			Locked string `json:"locked"`
		} `json:"balances"`
	}

	if err := b.signedRequest("GET", endpoint, params, &result); err != nil {
		return nil, err
	}

	balances := make([]Balance, 0)
	for _, b := range result.Balances {
		free, _ := strconv.ParseFloat(b.Free, 64)
		locked, _ := strconv.ParseFloat(b.Locked, 64)

		// Only return non-zero balances
		if free > 0 || locked > 0 {
			balances = append(balances, Balance{
				Asset:  b.Asset,
				Free:   free,
				Locked: locked,
			})
		}
	}

	return balances, nil
}

// GetAssetBalance returns balance for a specific asset.
func (b *BinanceConnector) GetAssetBalance(asset string) (*Balance, error) {
	balances, err := b.GetBalances()
	if err != nil {
		return nil, err
	}

	for _, bal := range balances {
		if bal.Asset == asset {
			return &bal, nil
		}
	}

	// Return zero balance if not found
	return &Balance{Asset: asset, Free: 0, Locked: 0}, nil
}

// ============================================================================
// INTERNAL HELPERS
// ============================================================================

type binanceOrderResponse struct {
	Symbol        string `json:"symbol"`
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	Price         string `json:"price"`
	OrigQty       string `json:"origQty"`
	ExecutedQty   string `json:"executedQty"`
	Status        string `json:"status"`
	Type          string `json:"type"`
	Side          string `json:"side"`
	TransactTime  int64  `json:"transactTime"`
}

func (r *binanceOrderResponse) toOrder() *Order {
	price, _ := strconv.ParseFloat(r.Price, 64)
	origQty, _ := strconv.ParseFloat(r.OrigQty, 64)
	execQty, _ := strconv.ParseFloat(r.ExecutedQty, 64)

	return &Order{
		ID:           strconv.FormatInt(r.OrderID, 10),
		Symbol:       r.Symbol,
		Side:         r.Side,
		Type:         r.Type,
		Quantity:     origQty,
		Price:        price,
		Status:       r.Status,
		ExecutedQty:  execQty,
		TransactTime: r.TransactTime,
	}
}

// publicRequest makes an unauthenticated request.
func (b *BinanceConnector) publicRequest(method, endpoint string, params url.Values, result interface{}) error {
	reqURL := b.baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return err
	}

	return b.doRequest(req, result)
}

// signedRequest makes an authenticated request with HMAC signature.
func (b *BinanceConnector) signedRequest(method, endpoint string, params url.Values, result interface{}) error {
	// Add timestamp
	params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))

	// Create signature
	queryString := params.Encode()
	signature := b.sign(queryString)
	params.Set("signature", signature)

	reqURL := b.baseURL + endpoint + "?" + params.Encode()

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return err
	}

	// Add API key header
	req.Header.Set("X-MBX-APIKEY", b.config.APIKey)

	return b.doRequest(req, result)
}

// sign creates HMAC SHA256 signature.
func (b *BinanceConnector) sign(message string) string {
	mac := hmac.New(sha256.New, []byte(b.config.APISecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// doRequest executes the HTTP request.
func (b *BinanceConnector) doRequest(req *http.Request, result interface{}) error {
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode >= 400 {
		var apiErr struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Msg != "" {
			return fmt.Errorf("Binance API error %d: %s", apiErr.Code, apiErr.Msg)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// parseFloat helper for interface{} to float64.
func parseFloat(v interface{}) float64 {
	switch val := v.(type) {
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case float64:
		return val
	default:
		return 0
	}
}
