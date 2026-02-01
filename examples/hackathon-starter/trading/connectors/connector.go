package connectors

// Candle represents OHLCV data.
type Candle struct {
	OpenTime  int64   `json:"open_time"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	CloseTime int64   `json:"close_time"`
}

// Order represents a trading order.
type Order struct {
	ID           string  `json:"id"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"` // "BUY" or "SELL"
	Type         string  `json:"type"` // "MARKET", "LIMIT"
	Quantity     float64 `json:"quantity"`
	Price        float64 `json:"price"`
	Status       string  `json:"status"`
	ExecutedQty  float64 `json:"executed_qty"`
	AvgPrice     float64 `json:"avg_price"`
	TransactTime int64   `json:"transact_time"`
}

// Balance represents account balance for an asset.
type Balance struct {
	Asset  string  `json:"asset"`
	Free   float64 `json:"free"`
	Locked float64 `json:"locked"`
}

// ExchangeConnector is the interface all exchange implementations must satisfy.
type ExchangeConnector interface {
	// Name returns the exchange name.
	Name() string

	// SupportedSymbols returns list of tradeable symbols.
	SupportedSymbols() []string

	// IsTestnet returns true if connected to testnet.
	IsTestnet() bool

	// ==================== Market Data ====================

	// GetPrice returns the current price for a symbol.
	GetPrice(symbol string) (float64, error)

	// GetCandles returns historical OHLCV candles.
	GetCandles(symbol string, interval string, limit int) ([]Candle, error)

	// GetOrderBook returns current bid/ask (simplified).
	GetBestBidAsk(symbol string) (bid, ask float64, err error)

	// ==================== Trading (Authenticated) ====================

	// PlaceMarketOrder places a market order.
	PlaceMarketOrder(symbol, side string, quantity float64) (*Order, error)

	// PlaceLimitOrder places a limit order.
	PlaceLimitOrder(symbol, side string, quantity, price float64) (*Order, error)

	// CancelOrder cancels an open order.
	CancelOrder(symbol, orderID string) error

	// GetOrder gets order status.
	GetOrder(symbol, orderID string) (*Order, error)

	// GetOpenOrders returns all open orders.
	GetOpenOrders(symbol string) ([]Order, error)

	// ==================== Account ====================

	// GetBalance returns account balances.
	GetBalances() ([]Balance, error)

	// GetAssetBalance returns balance for a specific asset.
	GetAssetBalance(asset string) (*Balance, error)
}
