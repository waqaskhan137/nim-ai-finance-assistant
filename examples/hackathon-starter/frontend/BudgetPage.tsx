import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import './BudgetPage.css'
import { useToast } from './Toast'

// Envelope from journey database
interface Envelope {
  name: string
  amount: number
  spent: number
  guardrail: string
  threshold: number
  category: string
  color: string
}

// Budget from journey database
interface JourneyBudget {
  exists: boolean
  user_id?: string
  period?: string
  start_date?: number
  envelopes?: Envelope[]
  buffer?: number
  total_budget?: number
  insights?: string[]
  created_at?: string
  updated_at?: string
  message?: string
}

// Analysis data from budget-plan endpoint
interface BudgetAnalysis {
  current_snapshot: {
    total_assets: number
    monthly_income: number
    monthly_expenses: number
    savings_rate_percent: number
  }
  spending_trackers: {
    daily: {
      today: { date: string; day_of_week: string; total_spent: number }
      daily_budget: number
      daily_average: number
      last_7_days: { date: string; day_of_week: string; total_spent: number }[]
      today_vs_budget: string
    }
    monthly: {
      current_month: { month: string; total_spent: number }
      monthly_budget: number
      days_remaining: number
      daily_budget_remaining: number
      projected_end_of_month: number
    }
  }
  spending_by_category: { category: string; amount: number; percentage: number }[]
  budget_score: number
  budget_grade: string
  summary_insight: string
}

const guardrailInfo: Record<string, { label: string; description: string; color: string }> = {
  hard: { label: 'Hard Stop', description: 'Alerts at 100%', color: '#FF3B30' },
  soft: { label: 'Soft Alert', description: 'Alerts at threshold', color: '#FF9500' },
  auto_pay: { label: 'Auto-Pay', description: 'Bills auto-deducted', color: '#007AFF' },
  protected: { label: 'Protected', description: "Can't spend from this", color: '#34C759' },
}

const defaultColors: Record<string, string> = {
  Needs: '#007AFF',
  Wants: '#AF52DE',
  Bills: '#FF9500',
  Goals: '#34C759',
}

export function BudgetPage() {
  const navigate = useNavigate()
  const toast = useToast()
  
  const [journeyBudget, setJourneyBudget] = useState<JourneyBudget | null>(null)
  const [analysis, setAnalysis] = useState<BudgetAnalysis | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  
  // Create budget form state
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [formIncome, setFormIncome] = useState('')
  const [formStartDate, setFormStartDate] = useState('1')
  const [creating, setCreating] = useState(false)

  // Edit mode
  const [editMode, setEditMode] = useState(false)
  const [editEnvelopes, setEditEnvelopes] = useState<Envelope[]>([])

  useEffect(() => { fetchData() }, [])

  const fetchData = async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    else setLoading(true)
    
    try {
      // Fetch both endpoints in parallel
      const [budgetRes, analysisRes] = await Promise.all([
        fetch('/api/budget'),
        fetch('/api/budget-plan')
      ])
      
      if (budgetRes.ok) {
        const budgetData = await budgetRes.json()
        setJourneyBudget(budgetData)
        if (budgetData.exists && budgetData.envelopes) {
          setEditEnvelopes(budgetData.envelopes)
        }
      }
      
      if (analysisRes.ok) {
        setAnalysis(await analysisRes.json())
      }
      
      if (isRefresh) toast.success('Refreshed', 'Budget data updated')
    } catch {
      toast.error('Error', 'Failed to load budget')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const createBudget = async () => {
    const income = parseFloat(formIncome)
    if (!income || income <= 0) {
      toast.error('Invalid', 'Please enter a valid income amount')
      return
    }

    setCreating(true)
    try {
      const res = await fetch('/api/budget', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          total_budget: income,
          period: 'monthly',
          start_date: parseInt(formStartDate),
          buffer: income * 0.05, // 5% buffer
        })
      })
      
      if (res.ok) {
        toast.success('Created!', 'Your budget has been set up')
        setShowCreateForm(false)
        fetchData()
      } else {
        toast.error('Error', 'Failed to create budget')
      }
    } catch {
      toast.error('Error', 'Failed to create budget')
    } finally {
      setCreating(false)
    }
  }

  const saveBudgetEdits = async () => {
    if (!journeyBudget?.total_budget) return
    
    setCreating(true)
    try {
      const res = await fetch('/api/budget', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          total_budget: journeyBudget.total_budget,
          period: journeyBudget.period,
          start_date: journeyBudget.start_date,
          buffer: journeyBudget.buffer,
          envelopes: editEnvelopes.map(e => ({
            name: e.name,
            amount: e.amount,
            guardrail: e.guardrail,
            threshold: e.threshold,
            category: e.category,
            color: e.color,
          }))
        })
      })
      
      if (res.ok) {
        toast.success('Saved!', 'Budget updated')
        setEditMode(false)
        fetchData()
      }
    } catch {
      toast.error('Error', 'Failed to save changes')
    } finally {
      setCreating(false)
    }
  }

  const updateEnvelopeAmount = (index: number, newAmount: number) => {
    const updated = [...editEnvelopes]
    updated[index] = { ...updated[index], amount: newAmount }
    setEditEnvelopes(updated)
  }

  const fmt = (n: number) => '$' + n.toLocaleString(undefined, { maximumFractionDigits: 0 })

  if (loading) {
    return (
      <div className="compact-page">
        <header className="compact-header">
          <Link to="/" className="back-btn">←</Link>
          <h1>Budget</h1>
        </header>
        <div className="loading-state">Loading...</div>
      </div>
    )
  }

  const hasBudget = journeyBudget?.exists && journeyBudget.envelopes && journeyBudget.envelopes.length > 0

  return (
    <div className="compact-page">
      <header className="compact-header">
        <div className="header-left">
          <Link to="/" className="back-btn">←</Link>
          <h1>Budget</h1>
        </div>
        <button className="refresh-btn" onClick={() => fetchData(true)} disabled={refreshing}>
          {refreshing ? '...' : '↻'}
        </button>
      </header>

      <div className="content-grid">
        {/* No Budget - Create Form */}
        {!hasBudget && !showCreateForm && (
          <div className="empty-state-card">
            <div className="empty-icon">📊</div>
            <h2>No Budget Yet</h2>
            <p>Create an envelope-based budget to track your spending and reach your goals.</p>
            <div className="empty-actions">
              <button className="primary-btn" onClick={() => setShowCreateForm(true)}>
                Create Budget
              </button>
              <button className="secondary-btn" onClick={() => navigate('/chat', { state: { autoMessage: 'Help me create a budget' } })}>
                Ask Nim to Help
              </button>
            </div>
          </div>
        )}

        {/* Create Budget Form */}
        {!hasBudget && showCreateForm && (
          <div className="create-budget-card">
            <h2>Create Your Budget</h2>
            <p className="form-description">We'll create a 50/20/20/10 budget split (Needs/Wants/Bills/Goals)</p>
            
            <div className="form-group">
              <label>Monthly Income</label>
              <input
                type="number"
                placeholder="e.g., 3000"
                value={formIncome}
                onChange={(e) => setFormIncome(e.target.value)}
                className="form-input"
              />
            </div>
            
            <div className="form-group">
              <label>Budget Start Day (Payday)</label>
              <select 
                value={formStartDate} 
                onChange={(e) => setFormStartDate(e.target.value)}
                className="form-select"
              >
                {Array.from({ length: 28 }, (_, i) => (
                  <option key={i + 1} value={i + 1}>{i + 1}</option>
                ))}
              </select>
            </div>

            {formIncome && parseFloat(formIncome) > 0 && (
              <div className="preview-envelopes">
                <h4>Preview</h4>
                <div className="preview-list">
                  <div className="preview-item"><span>Needs</span><span>{fmt(parseFloat(formIncome) * 0.5)}</span></div>
                  <div className="preview-item"><span>Wants</span><span>{fmt(parseFloat(formIncome) * 0.2)}</span></div>
                  <div className="preview-item"><span>Bills</span><span>{fmt(parseFloat(formIncome) * 0.2)}</span></div>
                  <div className="preview-item"><span>Goals</span><span>{fmt(parseFloat(formIncome) * 0.1)}</span></div>
                </div>
              </div>
            )}

            <div className="form-actions">
              <button className="secondary-btn" onClick={() => setShowCreateForm(false)}>Cancel</button>
              <button className="primary-btn" onClick={createBudget} disabled={creating || !formIncome}>
                {creating ? 'Creating...' : 'Create Budget'}
              </button>
            </div>
          </div>
        )}

        {/* Budget Exists - Show Envelopes */}
        {hasBudget && (
          <>
            {/* Budget Header */}
            <div className="budget-summary-card">
              <div className="budget-summary-row">
                <div className="budget-summary-item">
                  <span className="summary-label">Total Budget</span>
                  <span className="summary-value">{fmt(journeyBudget.total_budget || 0)}</span>
                </div>
                <div className="budget-summary-item">
                  <span className="summary-label">Period</span>
                  <span className="summary-value">{journeyBudget.period}</span>
                </div>
                <div className="budget-summary-item">
                  <span className="summary-label">Starts</span>
                  <span className="summary-value">Day {journeyBudget.start_date}</span>
                </div>
              </div>
            </div>

            {/* Envelopes */}
            <div className="section-card">
              <div className="section-header">
                <h3>Envelopes</h3>
                <div className="section-actions">
                  {!editMode ? (
                    <button className="edit-link" onClick={() => setEditMode(true)}>Edit</button>
                  ) : (
                    <>
                      <button className="cancel-link" onClick={() => { setEditMode(false); setEditEnvelopes(journeyBudget?.envelopes || []) }}>Cancel</button>
                      <button className="save-link" onClick={saveBudgetEdits} disabled={creating}>
                        {creating ? '...' : 'Save'}
                      </button>
                    </>
                  )}
                </div>
              </div>
              <div className="envelopes-list">
                {(editMode ? editEnvelopes : journeyBudget.envelopes)?.map((env, idx) => {
                  const spent = env.spent || 0
                  const percentUsed = env.amount > 0 ? (spent / env.amount) * 100 : 0
                  const remaining = env.amount - spent
                  const isOver = percentUsed > 100
                  const isWarning = percentUsed >= (env.threshold * 100)
                  const color = env.color || defaultColors[env.name] || '#007AFF'
                  const guardrail = guardrailInfo[env.guardrail] || guardrailInfo.soft

                  return (
                    <div key={env.name} className={`envelope-card ${isOver ? 'over' : isWarning ? 'warning' : ''}`}>
                      <div className="envelope-header">
                        <div className="envelope-name" style={{ color }}>
                          {env.name}
                        </div>
                        <div className="envelope-guardrail" style={{ background: guardrail.color }}>
                          {guardrail.label}
                        </div>
                      </div>
                      
                      {editMode ? (
                        <div className="envelope-edit">
                          <input
                            type="number"
                            value={editEnvelopes[idx]?.amount || 0}
                            onChange={(e) => updateEnvelopeAmount(idx, parseFloat(e.target.value) || 0)}
                            className="envelope-amount-input"
                          />
                        </div>
                      ) : (
                        <>
                          <div className="envelope-progress">
                            <div className="progress-bar">
                              <div 
                                className="progress-fill"
                                style={{ 
                                  width: `${Math.min(percentUsed, 100)}%`,
                                  background: isOver ? '#FF3B30' : isWarning ? '#FF9500' : color
                                }}
                              />
                            </div>
                          </div>
                          <div className="envelope-amounts">
                            <span className="spent">{fmt(spent)} spent</span>
                            <span className="allocated">{fmt(env.amount)} allocated</span>
                          </div>
                          <div className="envelope-remaining">
                            {isOver ? (
                              <span className="over-budget">Over by {fmt(Math.abs(remaining))}</span>
                            ) : (
                              <span className="under-budget">{fmt(remaining)} remaining</span>
                            )}
                          </div>
                        </>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>

            {/* Insights */}
            {journeyBudget.insights && journeyBudget.insights.length > 0 && (
              <div className="section-card">
                <h3>Insights</h3>
                <ul className="insights-list">
                  {journeyBudget.insights.map((insight, i) => (
                    <li key={i}>{insight}</li>
                  ))}
                </ul>
              </div>
            )}
          </>
        )}

        {/* Analysis Section (always show if available) */}
        {analysis && (
          <>
            {/* Score Banner */}
            <div className={`score-banner ${analysis.budget_score >= 70 ? 'good' : analysis.budget_score >= 50 ? 'ok' : 'bad'}`}>
              <div className="score-main">
                <span className="score-num">{analysis.budget_score}</span>
                <span className="score-grade">{analysis.budget_grade}</span>
              </div>
              <p className="score-insight">{analysis.summary_insight}</p>
            </div>

            {/* Quick Stats */}
            <div className="stats-row">
              <div className="stat-item">
                <span className="stat-value">{fmt(analysis.current_snapshot.total_assets)}</span>
                <span className="stat-label">Assets</span>
              </div>
              <div className="stat-item">
                <span className="stat-value positive">+{fmt(analysis.current_snapshot.monthly_income)}</span>
                <span className="stat-label">Income</span>
              </div>
              <div className="stat-item">
                <span className="stat-value negative">-{fmt(analysis.current_snapshot.monthly_expenses)}</span>
                <span className="stat-label">Expenses</span>
              </div>
              <div className="stat-item">
                <span className={`stat-value ${analysis.current_snapshot.savings_rate_percent >= 0 ? 'positive' : 'negative'}`}>
                  {analysis.current_snapshot.savings_rate_percent.toFixed(0)}%
                </span>
                <span className="stat-label">Saved</span>
              </div>
            </div>

            {/* Today vs Budget */}
            <div className="budget-card">
              <div className="budget-card-header">
                <span>Today</span>
                <span className={`status-pill ${analysis.spending_trackers.daily.today_vs_budget}`}>
                  {analysis.spending_trackers.daily.today_vs_budget === 'under' ? 'Under' : 
                   analysis.spending_trackers.daily.today_vs_budget === 'over' ? 'Over' : 'On Track'}
                </span>
              </div>
              <div className="budget-progress">
                <div className="progress-bar">
                  <div 
                    className={`progress-fill ${analysis.spending_trackers.daily.today.total_spent > analysis.spending_trackers.daily.daily_budget ? 'over' : ''}`}
                    style={{ width: `${Math.min((analysis.spending_trackers.daily.today.total_spent / analysis.spending_trackers.daily.daily_budget) * 100, 100)}%` }}
                  />
                </div>
                <div className="progress-labels">
                  <span>{fmt(analysis.spending_trackers.daily.today.total_spent)} spent</span>
                  <span>{fmt(analysis.spending_trackers.daily.daily_budget)} budget</span>
                </div>
              </div>
            </div>

            {/* Categories */}
            <div className="section-card">
              <h3>Spending by Category</h3>
              <div className="category-list">
                {analysis.spending_by_category.slice(0, 6).map(cat => (
                  <div key={cat.category} className="category-row">
                    <span className="cat-name">{cat.category.replace('_', ' ')}</span>
                    <div className="cat-bar">
                      <div className="cat-fill" style={{ width: `${cat.percentage}%` }} />
                    </div>
                    <span className="cat-amount">{fmt(cat.amount)}</span>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
