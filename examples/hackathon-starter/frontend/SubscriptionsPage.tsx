import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import './SubscriptionsPage.css'
import { useToast } from './Toast'

interface Subscription {
  id: string
  name: string
  category: string
  amount: number
  frequency: string
  status: string
  icon: string
  color: string
  usage_insight: string
}

interface CategoryCost {
  category: string
  monthly_total: number
  count: number
  percentage: number
  icon: string
  color: string
}

interface Recommendation {
  type: string
  priority: string
  title: string
  savings?: number
}

interface SubscriptionAnalysis {
  subscriptions: Subscription[]
  total_monthly: number
  total_yearly: number
  subscription_count: number
  category_breakdown: CategoryCost[]
  potential_savings: number
  forgotten_subscriptions: Subscription[]
  recommendations: Recommendation[]
  health_score: number
  health_grade: string
  summary_insight: string
}

export function SubscriptionsPage() {
  const [data, setData] = useState<SubscriptionAnalysis | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [filter, setFilter] = useState<'all' | 'forgotten'>('all')
  const [expandedSub, setExpandedSub] = useState<string | null>(null)
  const toast = useToast()
  const navigate = useNavigate()
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081'

  // Navigate to chat with pre-filled message
  const askNim = (action: 'cancel' | 'review' | 'alternatives', subName: string) => {
    const messages: Record<string, string> = {
      cancel: `Help me cancel my ${subName} subscription`,
      review: `Should I keep my ${subName} subscription? Analyze my usage.`,
      alternatives: `What are some cheaper alternatives to ${subName}?`
    }
    navigate('/chat', { state: { autoMessage: messages[action] } })
  }

  useEffect(() => { fetchData() }, [])

  const fetchData = async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    else setLoading(true)
    try {
      const res = await fetch(`${apiUrl}/api/subscriptions`)
      if (res.ok) setData(await res.json())
      if (isRefresh) toast.success('Refreshed', 'Subscription data updated')
    } catch {
      toast.error('Error', 'Failed to load subscriptions')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const fmt = (n: number) => '$' + n.toLocaleString(undefined, { maximumFractionDigits: 0 })

  const freqLabel = (f: string) => {
    const labels: Record<string, string> = {
      weekly: '/wk', monthly: '/mo', yearly: '/yr', quarterly: '/qtr'
    }
    return labels[f] || ''
  }

  if (loading) return (
    <div className="compact-page">
      <header className="compact-header">
        <div className="header-left">
          <Link to="/" className="back-btn">←</Link>
          <h1>Subscriptions</h1>
        </div>
      </header>
      <div className="loading-state">
        <div className="loading-spinner"></div>
        <span>Analyzing subscriptions...</span>
      </div>
    </div>
  )

  if (!data) return null

  const subs = filter === 'forgotten' ? data.forgotten_subscriptions : data.subscriptions

  return (
    <div className="compact-page">
      <header className="compact-header">
        <div className="header-left">
          <Link to="/" className="back-btn">←</Link>
          <h1>Subscriptions</h1>
        </div>
        <button className="refresh-btn" onClick={() => fetchData(true)} disabled={refreshing}>
          {refreshing ? <span className="btn-spinner"></span> : null}
          {refreshing ? 'Refreshing' : 'Refresh'}
        </button>
      </header>

      <div className="content-grid">
        {/* Score Banner */}
        <div className={`score-banner ${data.health_score >= 70 ? 'good' : data.health_score >= 50 ? 'ok' : 'bad'}`}>
          <div className="score-main">
            <span className="score-num">{data.health_score}</span>
            <span className="score-grade">{data.health_grade}</span>
          </div>
          <p className="score-insight">{data.summary_insight}</p>
        </div>

        {/* Quick Stats */}
        <div className="stats-row">
          <div className="stat-item">
            <span className="stat-value">{data.subscription_count}</span>
            <span className="stat-label">Active</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{fmt(data.total_monthly)}</span>
            <span className="stat-label">Monthly</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{fmt(data.total_yearly)}</span>
            <span className="stat-label">Yearly</span>
          </div>
          <div className="stat-item highlight">
            <span className="stat-value positive">{fmt(data.potential_savings)}</span>
            <span className="stat-label">Can Save</span>
          </div>
        </div>

        {/* Quick Recommendations */}
        {data.recommendations.length > 0 && (
          <div className="recs-banner">
            {data.recommendations.slice(0, 3).map((rec, i) => (
              <button 
                key={i} 
                className={`rec-pill ${rec.priority} clickable`}
                onClick={() => navigate('/chat', { 
                  state: { autoMessage: rec.type === 'cancel' 
                    ? `Help me ${rec.title.toLowerCase()}` 
                    : `${rec.title} - what should I do?` 
                  } 
                })}
              >
                <span>{rec.type === 'cancel' ? '🗑️' : '👀'}</span>
                <span>{rec.title}</span>
                {rec.savings && <span className="rec-save">Save {fmt(rec.savings)}</span>}
                <span className="rec-arrow">→</span>
              </button>
            ))}
          </div>
        )}

        {/* Filter Tabs */}
        <div className="filter-tabs">
          <button className={`filter-btn ${filter === 'all' ? 'active' : ''}`} onClick={() => setFilter('all')}>
            All ({data.subscription_count})
          </button>
          {data.forgotten_subscriptions.length > 0 && (
            <button className={`filter-btn warning ${filter === 'forgotten' ? 'active' : ''}`} onClick={() => setFilter('forgotten')}>
              Forgotten ({data.forgotten_subscriptions.length})
            </button>
          )}
        </div>

        {/* Subscriptions List */}
        <div className="subs-list">
          {subs.map(sub => (
            <div 
              key={sub.id} 
              className={`sub-row ${sub.status !== 'active' ? 'inactive' : ''} ${expandedSub === sub.id ? 'expanded' : ''}`}
              onClick={() => setExpandedSub(expandedSub === sub.id ? null : sub.id)}
            >
              <div className="sub-icon" style={{ background: sub.color + '20', color: sub.color }}>
                {sub.icon}
              </div>
              <div className="sub-info">
                <span className="sub-name">{sub.name}</span>
                <span className="sub-meta">{sub.category} • {sub.usage_insight}</span>
              </div>
              <div className="sub-cost">
                <span className="sub-amount">{fmt(sub.amount)}</span>
                <span className="sub-freq">{freqLabel(sub.frequency)}</span>
              </div>
              <span className="sub-chevron">{expandedSub === sub.id ? '▼' : '›'}</span>
              
              {/* Expanded Actions */}
              {expandedSub === sub.id && (
                <div className="sub-actions" onClick={e => e.stopPropagation()}>
                  <button 
                    className="sub-action-btn cancel"
                    onClick={() => askNim('cancel', sub.name)}
                  >
                    Cancel
                  </button>
                  <button 
                    className="sub-action-btn review"
                    onClick={() => askNim('review', sub.name)}
                  >
                    Review
                  </button>
                  <button 
                    className="sub-action-btn alternatives"
                    onClick={() => askNim('alternatives', sub.name)}
                  >
                    Alternatives
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>

        {/* Category Breakdown - spans both columns */}
        <div className="section-card" style={{ gridColumn: '1 / -1' }}>
          <h3>By Category</h3>
          <div className="category-list">
            {data.category_breakdown.map(cat => (
              <div key={cat.category} className="category-row">
                <span className="cat-icon">{cat.icon}</span>
                <span className="cat-name">{cat.category}</span>
                <div className="cat-bar">
                  <div className="cat-fill" style={{ width: `${cat.percentage}%`, background: cat.color }} />
                </div>
                <span className="cat-amount">{fmt(cat.monthly_total)}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Ask Nim CTA */}
        <div className="ask-nim-cta" style={{ gridColumn: '1 / -1' }}>
          <div className="cta-content">
            <span className="cta-icon">💬</span>
            <div className="cta-text">
              <strong>Need help managing subscriptions?</strong>
              <span>Nim can analyze usage, find savings, and help you cancel.</span>
            </div>
          </div>
          <button 
            className="cta-button"
            onClick={() => navigate('/chat', { 
              state: { autoMessage: 'Analyze all my subscriptions and tell me which ones I should cancel to save money' } 
            })}
          >
            Ask Nim
          </button>
        </div>
      </div>
    </div>
  )
}
