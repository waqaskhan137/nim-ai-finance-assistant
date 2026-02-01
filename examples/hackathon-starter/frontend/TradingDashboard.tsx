import { useEffect, useState } from 'react';

interface Position {
  id: string;
  symbol: string;
  side: string;
  entry_price: string;
  current_price: string;
  quantity: string;
  unrealized_pnl: string;
  stop_loss: string;
  take_profit: string;
}

interface TradingStatus {
  status: string;
  initial_budget: string;
  stop_loss_floor: string;
  current_cash: string;
  total_value: string;
  total_pnl: string;
  pnl_percent: string;
  distance_to_floor: string;
  risk_profile: string;
  open_positions: number;
  positions: Position[];
  trades_count: number;
  win_count: number;
  loss_count: number;
  win_rate: string;
}

export function TradingDashboard() {
  const [status, setStatus] = useState<TradingStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isExpanded, setIsExpanded] = useState(true);

  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081';

  const fetchStatus = async () => {
    try {
      const response = await fetch(`${apiUrl}/api/trading-status`);
      if (!response.ok) throw new Error('Failed to fetch');
      const data = await response.json();
      setStatus(data);
      setError(null);
    } catch (err) {
      setError('Unable to connect to trading server');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    const interval = setInterval(fetchStatus, 3000); // Poll every 3 seconds
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return (
      <div className="trading-dashboard loading">
        <div className="dashboard-header">
          <span className="dashboard-icon">📊</span>
          <span>Loading trading status...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="trading-dashboard error">
        <div className="dashboard-header">
          <span className="dashboard-icon">⚠️</span>
          <span>{error}</span>
        </div>
      </div>
    );
  }

  if (!status) return null;

  const pnl = parseFloat(status.total_pnl);
  const pnlClass = pnl > 0 ? 'positive' : pnl < 0 ? 'negative' : '';
  const statusClass = status.status === 'active' ? 'active' : 
                       status.status.includes('WARNING') ? 'warning' : 'stopped';

  return (
    <div className={`trading-dashboard ${isExpanded ? 'expanded' : 'collapsed'}`}>
      <div className="dashboard-header" onClick={() => setIsExpanded(!isExpanded)}>
        <div className="header-left">
          <span className="dashboard-icon">📊</span>
          <span className="dashboard-title">Trading Portal</span>
          <span className={`status-badge ${statusClass}`}>{status.status}</span>
        </div>
        <div className="header-right">
          <span className={`pnl ${pnlClass}`}>{status.pnl_percent}</span>
          <span className="toggle-icon">{isExpanded ? '▼' : '▶'}</span>
        </div>
      </div>

      {isExpanded && (
        <div className="dashboard-content">
          {/* Portfolio Overview */}
          <div className="section portfolio-overview">
            <div className="stat-grid">
              <div className="stat">
                <span className="stat-label">Portfolio Value</span>
                <span className="stat-value">${status.total_value}</span>
              </div>
              <div className="stat">
                <span className="stat-label">Available Cash</span>
                <span className="stat-value">${status.current_cash}</span>
              </div>
              <div className="stat">
                <span className="stat-label">Total P&L</span>
                <span className={`stat-value ${pnlClass}`}>${status.total_pnl}</span>
              </div>
              <div className="stat">
                <span className="stat-label">To Floor</span>
                <span className="stat-value">${status.distance_to_floor}</span>
              </div>
            </div>
          </div>

          {/* Risk Info */}
          <div className="section risk-info">
            <div className="risk-bar">
              <span className="risk-label">Risk Profile: <strong>{status.risk_profile}</strong></span>
              <span className="risk-label">Floor: <strong>${status.stop_loss_floor}</strong></span>
            </div>
          </div>

          {/* Open Positions */}
          <div className="section positions">
            <div className="section-header">
              <span>Open Positions ({status.open_positions})</span>
            </div>
            {status.positions && status.positions.length > 0 ? (
              <div className="positions-list">
                {status.positions.map((pos) => {
                  const positionPnl = parseFloat(pos.unrealized_pnl);
                  const positionPnlClass = positionPnl > 0 ? 'positive' : positionPnl < 0 ? 'negative' : '';
                  return (
                    <div key={pos.id} className="position-card">
                      <div className="position-header">
                        <span className={`position-side ${pos.side}`}>{pos.side.toUpperCase()}</span>
                        <span className="position-symbol">{pos.symbol}</span>
                        <span className={`position-pnl ${positionPnlClass}`}>${pos.unrealized_pnl}</span>
                      </div>
                      <div className="position-details">
                        <span>Entry: ${pos.entry_price}</span>
                        <span>Current: ${pos.current_price}</span>
                        <span>Qty: {pos.quantity}</span>
                      </div>
                      <div className="position-levels">
                        <span className="stop-loss">SL: ${pos.stop_loss}</span>
                        <span className="take-profit">TP: ${pos.take_profit}</span>
                      </div>
                    </div>
                  );
                })}
              </div>
            ) : (
              <div className="no-positions">No open positions</div>
            )}
          </div>

          {/* Stats */}
          <div className="section stats">
            <span>Trades: {status.trades_count}</span>
            <span>Wins: {status.win_count}</span>
            <span>Losses: {status.loss_count}</span>
            <span>Win Rate: {status.win_rate}</span>
          </div>
        </div>
      )}
    </div>
  );
}

export default TradingDashboard;
