import { useState, useEffect, useRef } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { createChart, ColorType, CandlestickData, Time, IChartApi, ISeriesApi, CandlestickSeries } from 'lightweight-charts'
import './TradingPage.css'

interface JourneyStatus {
  step_5_investing: boolean
  next_step: string
  next_step_number: number
  emergency_fund?: {
    current_stage: number
    target_stage_1: number
    target_stage_2: number
    target_stage_3: number
    current_amount: number
    stage_1_progress: number
    stage_2_progress: number
    stage_3_progress: number
  }
}

interface Position {
  id: string
  symbol: string
  side: string
  entry_price: number
  quantity: number
  current_price: number
  unrealized_pnl: number
  pnl_percent: number
  stop_loss?: number
  take_profit?: number
}

interface TradingStatus {
  status: string
  total_value: number
  current_cash: number
  initial_budget: number
  stop_loss_floor: number
  total_pnl: number
  open_positions: number
  positions: Position[]
  risk_profile: {
    name: string
    max_position_pct: number
    stop_loss_pct: number
    take_profit_pct: number
  }
}

interface PositionModal {
  show: boolean
  position: Position | null
  livePrice: number
  livePnl: number
  livePnlPercent: number
}

interface SymbolInfo {
  symbol: string
  displayName: string
  price: number
  change: number
  changePercent: number
}

const SYMBOLS = [
  { symbol: 'BTCUSDT', displayName: 'BTC/USDT' },
  { symbol: 'ETHUSDT', displayName: 'ETH/USDT' },
  { symbol: 'SOLUSDT', displayName: 'SOL/USDT' },
  { symbol: 'BNBUSDT', displayName: 'BNB/USDT' },
  { symbol: 'XRPUSDT', displayName: 'XRP/USDT' },
  { symbol: 'ADAUSDT', displayName: 'ADA/USDT' },
  { symbol: 'DOGEUSDT', displayName: 'DOGE/USDT' },
  { symbol: 'AVAXUSDT', displayName: 'AVAX/USDT' },
]

const TIMEFRAMES = [
  { label: '1m', interval: '1m', limit: 100 },
  { label: '5m', interval: '5m', limit: 100 },
  { label: '15m', interval: '15m', limit: 100 },
  { label: '1H', interval: '1h', limit: 100 },
  { label: '4H', interval: '4h', limit: 100 },
  { label: '1D', interval: '1d', limit: 100 },
]

export function TradingPage() {
  const navigate = useNavigate()
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)

  const [journeyStatus, setJourneyStatus] = useState<JourneyStatus | null>(null)
  const [journeyLoading, setJourneyLoading] = useState(true)
  const [status, setStatus] = useState<TradingStatus | null>(null)
  const [selectedSymbol, setSelectedSymbol] = useState('BTCUSDT')
  const [selectedTimeframe, setSelectedTimeframe] = useState('15m')
  const [symbolPrices, setSymbolPrices] = useState<Map<string, SymbolInfo>>(new Map())
  const [currentPrice, setCurrentPrice] = useState<number>(0)
  const [priceChange, setPriceChange] = useState<number>(0)
  const [positionModal, setPositionModal] = useState<PositionModal>({
    show: false,
    position: null,
    livePrice: 0,
    livePnl: 0,
    livePnlPercent: 0
  })
  const [closingPosition, setClosingPosition] = useState(false)

  // Fetch journey status to check if investing is unlocked
  useEffect(() => {
    const fetchJourneyStatus = async () => {
      try {
        const response = await fetch('/api/journey-status')
        if (response.ok) {
          const data = await response.json()
          setJourneyStatus(data)
        }
      } catch (err) {
        console.error('Failed to fetch journey status:', err)
      } finally {
        setJourneyLoading(false)
      }
    }

    fetchJourneyStatus()
  }, [])

  // Check if investing is unlocked
  const isInvestingUnlocked = journeyStatus?.step_5_investing ?? false

  // Fetch trading status
  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const response = await fetch('/api/trading-status')
        if (response.ok) {
          const data = await response.json()
          setStatus(data)
        }
      } catch (err) {
        console.error('Failed to fetch status:', err)
      }
    }

    fetchStatus()
    // Poll more frequently for real-time feel
    const interval = setInterval(fetchStatus, 2000)
    return () => clearInterval(interval)
  }, [])

  // Fetch live prices for watchlist
  useEffect(() => {
    const fetchPrices = async () => {
      try {
        const promises = SYMBOLS.map(async ({ symbol, displayName }) => {
          const res = await fetch(`https://api.binance.com/api/v3/ticker/24hr?symbol=${symbol}`)
          const data = await res.json()
          return {
            symbol,
            displayName,
            price: parseFloat(data.lastPrice),
            change: parseFloat(data.priceChange),
            changePercent: parseFloat(data.priceChangePercent),
          }
        })
        const results = await Promise.all(promises)
        const priceMap = new Map<string, SymbolInfo>()
        results.forEach(r => priceMap.set(r.symbol, r))
        setSymbolPrices(priceMap)

        const selected = priceMap.get(selectedSymbol)
        if (selected) {
          setCurrentPrice(selected.price)
          setPriceChange(selected.changePercent)
        }
      } catch (err) {
        console.error('Failed to fetch prices:', err)
      }
    }

    fetchPrices()
    const interval = setInterval(fetchPrices, 5000)
    return () => clearInterval(interval)
  }, [selectedSymbol])

  // Initialize chart
  useEffect(() => {
    if (journeyLoading || !isInvestingUnlocked || chartRef.current) return
    if (!chartContainerRef.current) return

    const chart = createChart(chartContainerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: '#0a0a0a' },
        textColor: '#d0d0d0',
      },
      grid: {
        vertLines: { color: '#1a1a1a' },
        horzLines: { color: '#1a1a1a' },
      },
      crosshair: {
        mode: 1,
        vertLine: { color: '#555', labelBackgroundColor: '#333' },
        horzLine: { color: '#555', labelBackgroundColor: '#333' },
      },
      rightPriceScale: {
        borderColor: '#1a1a1a',
        textColor: '#d0d0d0',
      },
      timeScale: {
        borderColor: '#1a1a1a',
        timeVisible: true,
        secondsVisible: false,
      },
      width: chartContainerRef.current.clientWidth,
      height: 400,
    })

    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#22c55e',
      downColor: '#ef4444',
      borderDownColor: '#ef4444',
      borderUpColor: '#22c55e',
      wickDownColor: '#ef4444',
      wickUpColor: '#22c55e',
    })

    chartRef.current = chart
    candleSeriesRef.current = candleSeries

    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ width: chartContainerRef.current.clientWidth })
      }
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
      chartRef.current = null
      candleSeriesRef.current = null
    }
  }, [isInvestingUnlocked, journeyLoading])

  // Fetch candlestick data
  useEffect(() => {
    const fetchCandles = async () => {
      try {
        const tf = TIMEFRAMES.find(t => t.label === selectedTimeframe) || TIMEFRAMES[2]
        const res = await fetch(
          `https://api.binance.com/api/v3/klines?symbol=${selectedSymbol}&interval=${tf.interval}&limit=${tf.limit}`
        )
        const data = await res.json()

        const candles: CandlestickData<Time>[] = data.map((d: number[]) => ({
          time: (d[0] / 1000) as Time,
          open: parseFloat(String(d[1])),
          high: parseFloat(String(d[2])),
          low: parseFloat(String(d[3])),
          close: parseFloat(String(d[4])),
        }))

        if (candleSeriesRef.current) {
          candleSeriesRef.current.setData(candles)
          chartRef.current?.timeScale().fitContent()
        }
      } catch (err) {
        console.error('Failed to fetch candles:', err)
      }
    }

    fetchCandles()
    const interval = setInterval(fetchCandles, 15000)
    return () => clearInterval(interval)
  }, [selectedSymbol, selectedTimeframe])

  // WebSocket for real-time updates
  useEffect(() => {
    const ws = new WebSocket(`wss://stream.binance.com:9443/ws/${selectedSymbol.toLowerCase()}@kline_1m`)

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.k) {
          const candle = data.k
          const newCandle: CandlestickData<Time> = {
            time: (candle.t / 1000) as Time,
            open: parseFloat(candle.o),
            high: parseFloat(candle.h),
            low: parseFloat(candle.l),
            close: parseFloat(candle.c),
          }

          if (candleSeriesRef.current) {
            candleSeriesRef.current.update(newCandle)
          }

          setCurrentPrice(parseFloat(candle.c))
        }
      } catch (err) {
        console.error('WebSocket error:', err)
      }
    }

    return () => ws.close()
  }, [selectedSymbol])

  const formatPrice = (price: number) => {
    if (price >= 1000) return price.toLocaleString(undefined, { maximumFractionDigits: 2 })
    if (price >= 1) return price.toFixed(2)
    return price.toFixed(6)
  }

  const pnlClass = (status?.total_pnl || 0) >= 0 ? 'positive' : 'negative'

  // Handle clicking on a position
  const openPositionModal = async (pos: Position) => {
    // Get live price for this symbol
    try {
      const res = await fetch(`https://api.binance.com/api/v3/ticker/price?symbol=${pos.symbol}`)
      const data = await res.json()
      const livePrice = parseFloat(data.price)
      
      // Calculate live P&L
      let livePnl: number
      if (pos.side === 'long') {
        livePnl = (livePrice - pos.entry_price) * pos.quantity
      } else {
        livePnl = (pos.entry_price - livePrice) * pos.quantity
      }
      const positionValue = pos.entry_price * pos.quantity
      const livePnlPercent = (livePnl / positionValue) * 100

      setPositionModal({
        show: true,
        position: pos,
        livePrice,
        livePnl,
        livePnlPercent
      })

      // Also switch chart to this symbol
      setSelectedSymbol(pos.symbol)
    } catch (err) {
      console.error('Failed to get live price:', err)
      setPositionModal({
        show: true,
        position: pos,
        livePrice: pos.current_price || pos.entry_price,
        livePnl: pos.unrealized_pnl,
        livePnlPercent: pos.pnl_percent
      })
    }
  }

  // Update modal P&L in real-time
  useEffect(() => {
    if (!positionModal.show || !positionModal.position) return

    const updateLivePnl = async () => {
      try {
        const pos = positionModal.position!
        const res = await fetch(`https://api.binance.com/api/v3/ticker/price?symbol=${pos.symbol}`)
        const data = await res.json()
        const livePrice = parseFloat(data.price)
        
        let livePnl: number
        if (pos.side === 'long') {
          livePnl = (livePrice - pos.entry_price) * pos.quantity
        } else {
          livePnl = (pos.entry_price - livePrice) * pos.quantity
        }
        const positionValue = pos.entry_price * pos.quantity
        const livePnlPercent = (livePnl / positionValue) * 100

        setPositionModal(prev => ({
          ...prev,
          livePrice,
          livePnl,
          livePnlPercent
        }))
      } catch (err) {
        // Ignore errors in background updates
      }
    }

    const interval = setInterval(updateLivePnl, 1000)
    return () => clearInterval(interval)
  }, [positionModal.show, positionModal.position?.id])

  // Close position via chat API
  const closePosition = async () => {
    if (!positionModal.position) return
    
    setClosingPosition(true)
    
    // Navigate to chat with a pre-filled message to close the position
    navigate('/chat', { 
      state: { 
        autoMessage: `Close my ${positionModal.position.symbol} position (ID: ${positionModal.position.id})` 
      } 
    })
  }

  const closeModal = () => {
    setPositionModal({ show: false, position: null, livePrice: 0, livePnl: 0, livePnlPercent: 0 })
    setClosingPosition(false)
  }

  // Show loading state
  if (journeyLoading) {
    return (
      <div className="trading-terminal">
        <header className="terminal-header">
          <div className="header-left">
            <Link to="/" className="back-btn">←</Link>
            <h1>Trading Terminal</h1>
          </div>
        </header>
        <div className="trading-gate-container">
          <div className="trading-gate-loading">Loading...</div>
        </div>
      </div>
    )
  }

  // Show locked state if investing isn't unlocked
  if (!isInvestingUnlocked) {
    const efProgress = journeyStatus?.emergency_fund?.stage_1_progress ?? 0
    const efCurrent = journeyStatus?.emergency_fund?.current_amount ?? 0
    const efTarget = journeyStatus?.emergency_fund?.target_stage_1 ?? 0

    return (
      <div className="trading-terminal">
        <header className="terminal-header">
          <div className="header-left">
            <Link to="/" className="back-btn">←</Link>
            <h1>Trading Terminal</h1>
          </div>
        </header>
        <div className="trading-gate-container">
          <div className="trading-gate">
            <div className="gate-icon">🔒</div>
            <h2 className="gate-title">Trading is Locked</h2>
            <p className="gate-description">
              Complete Stage 1 of your Emergency Fund before investing.
              This ensures you have a financial safety net (2 weeks of expenses)
              so you can invest with confidence.
            </p>
            
            <div className="gate-progress-section">
              <div className="gate-progress-header">
                <span>Emergency Fund Progress</span>
                <span>{efProgress.toFixed(0)}%</span>
              </div>
              <div className="gate-progress-bar">
                <div 
                  className="gate-progress-fill" 
                  style={{ width: `${Math.min(efProgress, 100)}%` }}
                />
              </div>
              <div className="gate-progress-amounts">
                <span>${efCurrent.toLocaleString()}</span>
                <span>of ${efTarget.toLocaleString()} (Stage 1)</span>
              </div>
            </div>

            <div className="gate-steps">
              <h3>Steps to Unlock Trading:</h3>
              <ol>
                <li className={journeyStatus?.step_5_investing ? 'completed' : ''}>
                  Create a budget to understand your expenses
                </li>
                <li className={journeyStatus?.step_5_investing ? 'completed' : ''}>
                  Set up savings rules to automate your savings
                </li>
                <li className={journeyStatus?.step_5_investing ? 'completed' : 'current'}>
                  Build 2 weeks of expenses in your Emergency Fund
                </li>
              </ol>
            </div>

            <div className="gate-actions">
              <Link to="/savings" className="gate-btn primary">
                Go to Smart Savings
              </Link>
              <Link to="/chat" className="gate-btn secondary">
                Chat with Nim
              </Link>
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="trading-terminal">
      {/* Position Detail Modal */}
      {positionModal.show && positionModal.position && (
        <div className="position-modal-overlay" onClick={closeModal}>
          <div className="position-modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <h3>{positionModal.position.symbol}</h3>
              <span className={`modal-side ${positionModal.position.side}`}>
                {positionModal.position.side.toUpperCase()}
              </span>
              <button className="modal-close" onClick={closeModal}>×</button>
            </div>
            
            <div className="modal-body">
              <div className="modal-stat">
                <span className="modal-label">Entry Price</span>
                <span className="modal-value">${formatPrice(positionModal.position.entry_price)}</span>
              </div>
              <div className="modal-stat">
                <span className="modal-label">Current Price</span>
                <span className={`modal-value live-price ${positionModal.livePnl >= 0 ? 'positive' : 'negative'}`}>
                  ${formatPrice(positionModal.livePrice)}
                  <span className="live-indicator">● LIVE</span>
                </span>
              </div>
              <div className="modal-stat">
                <span className="modal-label">Quantity</span>
                <span className="modal-value">{positionModal.position.quantity.toFixed(6)}</span>
              </div>
              <div className="modal-stat">
                <span className="modal-label">Position Value</span>
                <span className="modal-value">
                  ${(positionModal.livePrice * positionModal.position.quantity).toFixed(2)}
                </span>
              </div>
              <div className="modal-stat highlight">
                <span className="modal-label">Unrealized P&L</span>
                <span className={`modal-value pnl ${positionModal.livePnl >= 0 ? 'positive' : 'negative'}`}>
                  {positionModal.livePnl >= 0 ? '+' : ''}${positionModal.livePnl.toFixed(4)}
                  <span className="pnl-pct">
                    ({positionModal.livePnlPercent >= 0 ? '+' : ''}{positionModal.livePnlPercent.toFixed(2)}%)
                  </span>
                </span>
              </div>
              {positionModal.position.stop_loss && (
                <div className="modal-stat">
                  <span className="modal-label">Stop Loss</span>
                  <span className="modal-value stop-loss">${formatPrice(positionModal.position.stop_loss)}</span>
                </div>
              )}
              {positionModal.position.take_profit && (
                <div className="modal-stat">
                  <span className="modal-label">Take Profit</span>
                  <span className="modal-value take-profit">${formatPrice(positionModal.position.take_profit)}</span>
                </div>
              )}
            </div>

            <div className="modal-actions">
              <button className="modal-btn cancel" onClick={closeModal}>
                Keep Open
              </button>
              <button 
                className="modal-btn close-position" 
                onClick={closePosition}
                disabled={closingPosition}
              >
                {closingPosition ? 'Closing...' : 'Close Position'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Header */}
      <header className="terminal-header">
        <div className="header-left">
          <Link to="/" className="back-btn">←</Link>
          <h1>Trading Terminal</h1>
        </div>
        <div className="header-right">
          <span className={`connection-status ${status ? 'online' : 'offline'}`}>
            {status ? '● Live' : 'Connecting...'}
          </span>
        </div>
      </header>

      <div className="terminal-content">
        {/* Left: Watchlist */}
        <aside className="watchlist">
          <div className="watchlist-header">Markets</div>
          <div className="watchlist-items">
            {SYMBOLS.map(({ symbol, displayName }) => {
              const info = symbolPrices.get(symbol)
              const isActive = selectedSymbol === symbol
              return (
                <div
                  key={symbol}
                  className={`watchlist-item ${isActive ? 'active' : ''}`}
                  onClick={() => setSelectedSymbol(symbol)}
                >
                  <span className="symbol-name">{displayName}</span>
                  <div className="symbol-price-info">
                    <span className="symbol-price">
                      ${info ? formatPrice(info.price) : '—'}
                    </span>
                    <span className={`symbol-change ${(info?.changePercent || 0) >= 0 ? 'positive' : 'negative'}`}>
                      {info ? `${info.changePercent >= 0 ? '+' : ''}${info.changePercent.toFixed(2)}%` : '—'}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        </aside>

        {/* Center: Chart */}
        <main className="chart-area">
          <div className="chart-header">
            <div className="chart-symbol">
              <span className="symbol-title">{SYMBOLS.find(s => s.symbol === selectedSymbol)?.displayName}</span>
              <span className={`symbol-current-price ${priceChange >= 0 ? 'positive' : 'negative'}`}>
                ${formatPrice(currentPrice)}
              </span>
              <span className={`symbol-change-badge ${priceChange >= 0 ? 'positive' : 'negative'}`}>
                {priceChange >= 0 ? '+' : ''}{priceChange.toFixed(2)}%
              </span>
            </div>
            <div className="timeframe-selector">
              {TIMEFRAMES.map(tf => (
                <button
                  key={tf.label}
                  className={`tf-btn ${selectedTimeframe === tf.label ? 'active' : ''}`}
                  onClick={() => setSelectedTimeframe(tf.label)}
                >
                  {tf.label}
                </button>
              ))}
            </div>
          </div>
          <div className="chart-container" ref={chartContainerRef} />
        </main>

        {/* Right: Portfolio & Positions */}
        <aside className="portfolio-panel">
          <div className="portfolio-section">
            <div className="section-title">Portfolio</div>
            {status ? (
              <div className="portfolio-stats">
                <div className="stat-row">
                  <span className="stat-label">Total Value</span>
                  <span className="stat-value">${status.total_value.toFixed(2)}</span>
                </div>
                <div className="stat-row">
                  <span className="stat-label">Available</span>
                  <span className="stat-value">${status.current_cash.toFixed(2)}</span>
                </div>
                <div className="stat-row">
                  <span className="stat-label">P&L</span>
                  <span className={`stat-value ${pnlClass}`}>
                    {status.total_pnl >= 0 ? '+' : ''}${status.total_pnl.toFixed(2)}
                  </span>
                </div>
                <div className="stat-row">
                  <span className="stat-label">Positions</span>
                  <span className="stat-value">{status.open_positions}</span>
                </div>
              </div>
            ) : (
              <div className="loading-placeholder">Loading...</div>
            )}
          </div>

          <div className="positions-section">
            <div className="section-title">Open Positions</div>
            <div className="positions-list">
              {status?.positions && status.positions.length > 0 ? (
                status.positions.slice(0, 10).map(pos => (
                  <div 
                    key={pos.id} 
                    className="position-item clickable"
                    onClick={() => openPositionModal(pos)}
                  >
                    <div className="position-info">
                      <span className="position-symbol">{pos.symbol}</span>
                      <span className={`position-side ${pos.side}`}>{pos.side.toUpperCase()}</span>
                    </div>
                    <div className="position-pnl">
                      <span className={`pnl-value ${pos.unrealized_pnl >= 0 ? 'positive' : 'negative'}`}>
                        {pos.unrealized_pnl >= 0 ? '+' : ''}${pos.unrealized_pnl.toFixed(4)}
                      </span>
                      <span className={`pnl-percent ${pos.pnl_percent >= 0 ? 'positive' : 'negative'}`}>
                        {pos.pnl_percent >= 0 ? '+' : ''}{pos.pnl_percent.toFixed(2)}%
                      </span>
                    </div>
                    <span className="position-arrow">›</span>
                  </div>
                ))
              ) : (
                <div className="no-positions">No open positions</div>
              )}
              {status?.positions && status.positions.length > 10 && (
                <div className="more-positions">
                  +{status.positions.length - 10} more positions
                </div>
              )}
            </div>
          </div>


        </aside>
      </div>
    </div>
  )
}
