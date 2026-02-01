import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import './SmartSavingsPage.css'
import { useToast } from './Toast'

// Savings Rule from journey database
interface SavingsRule {
  id: string
  name: string
  type: string
  trigger: string
  action: string
  amount: number
  percentage: number
  destination: string
  active: boolean
  total_saved: number
  run_count: number
}

// Emergency Fund Stage
interface EmergencyFundStage {
  stage: number
  name: string
  target: number
  progress: number
  complete: boolean
  description: string
}

// Emergency Fund from journey database
interface EmergencyFund {
  exists: boolean
  current: number
  stage_1_target: number
  stage_2_target: number
  stage_3_target: number
  current_stage: number
  investing_unlocked: boolean
  stages: EmergencyFundStage[]
}

// Savings Rules response
interface SavingsRulesResponse {
  success: boolean
  rules: SavingsRule[]
  total_rules: number
  total_saved: number
}

// Analysis data
interface SavingsAnalysis {
  wallet_balance: number
  total_savings: number
  idle_cash: number
  current_weighted_apy: number
  monthly_earnings: number
  yearly_earnings: number
  has_opportunity: boolean
  sweep_amount: number
  projected_gain_yearly: number
  savings_health_score: number
  savings_health_grade: string
  summary_insight: string
  best_vault_for_user?: { name: string; apy: number }
}

const triggerLabels: Record<string, string> = {
  income_detected: 'After Payday',
  purchase: 'Every Purchase',
  budget_under: 'Under Budget',
  weekly: 'Weekly',
  monthly: 'Monthly',
}

const destinationLabels: Record<string, string> = {
  savings_emergency: 'Emergency Fund',
  savings_goals: 'Goals',
  investment: 'Investment',
}

export function SmartSavingsPage() {
  const navigate = useNavigate()
  const toast = useToast()
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081'

  const [analysis, setAnalysis] = useState<SavingsAnalysis | null>(null)
  const [rules, setRules] = useState<SavingsRule[]>([])
  const [totalSaved, setTotalSaved] = useState(0)
  const [emergencyFund, setEmergencyFund] = useState<EmergencyFund | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  // Create rule form
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [creating, setCreating] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    type: 'scheduled',
    trigger: 'monthly',
    action: 'transfer',
    amount: '',
    percentage: '',
    destination: 'savings_emergency',
  })

  useEffect(() => { fetchData() }, [])

  const fetchData = async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true)
    else setLoading(true)

    try {
      const [analysisRes, rulesRes, efRes] = await Promise.all([
        fetch(`${apiUrl}/api/savings-analysis?risk_tolerance=low`),
        fetch(`${apiUrl}/api/savings-rules`),
        fetch(`${apiUrl}/api/emergency-fund`),
      ])

      if (analysisRes.ok) setAnalysis(await analysisRes.json())
      
      if (rulesRes.ok) {
        const rulesData: SavingsRulesResponse = await rulesRes.json()
        setRules(rulesData.rules || [])
        setTotalSaved(rulesData.total_saved || 0)
      }

      if (efRes.ok) setEmergencyFund(await efRes.json())

      if (isRefresh) toast.success('Refreshed', 'Savings data updated')
    } catch {
      toast.error('Error', 'Failed to load savings')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const createRule = async () => {
    if (!formData.name) {
      toast.error('Required', 'Please enter a rule name')
      return
    }

    setCreating(true)
    try {
      const res = await fetch(`${apiUrl}/api/savings-rules`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: formData.name,
          type: formData.type,
          trigger: formData.trigger,
          action: formData.action,
          amount: parseFloat(formData.amount) || 0,
          percentage: parseFloat(formData.percentage) / 100 || 0,
          destination: formData.destination,
        })
      })

      if (res.ok) {
        toast.success('Created!', 'Savings rule is now active')
        setShowCreateForm(false)
        setFormData({
          name: '',
          type: 'scheduled',
          trigger: 'monthly',
          action: 'transfer',
          amount: '',
          percentage: '',
          destination: 'savings_emergency',
        })
        fetchData()
      }
    } catch {
      toast.error('Error', 'Failed to create rule')
    } finally {
      setCreating(false)
    }
  }

  const fmt = (n: number) => '$' + n.toLocaleString(undefined, { maximumFractionDigits: 0 })
  const pct = (n: number) => n.toFixed(1) + '%'

  if (loading) {
    return (
      <div className="compact-page">
        <header className="compact-header">
          <Link to="/" className="back-btn">←</Link>
          <h1>Smart Savings</h1>
        </header>
        <div className="loading-state">Loading...</div>
      </div>
    )
  }

  return (
    <div className="compact-page">
      <header className="compact-header">
        <div className="header-left">
          <Link to="/" className="back-btn">←</Link>
          <h1>Smart Savings</h1>
        </div>
        <button className="refresh-btn" onClick={() => fetchData(true)} disabled={refreshing}>
          {refreshing ? '...' : '↻'}
        </button>
      </header>

      <div className="content-grid">
        {/* Score Banner */}
        {analysis && (
          <div className={`score-banner ${analysis.savings_health_score >= 70 ? 'good' : analysis.savings_health_score >= 50 ? 'ok' : 'bad'}`}>
            <div className="score-main">
              <span className="score-num">{analysis.savings_health_score}</span>
              <span className="score-grade">{analysis.savings_health_grade}</span>
            </div>
            <p className="score-insight">{analysis.summary_insight}</p>
          </div>
        )}

        {/* Quick Stats */}
        {analysis && (
          <div className="stats-row">
            <div className="stat-item">
              <span className="stat-value">{fmt(analysis.total_savings)}</span>
              <span className="stat-label">Saved</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">{pct(analysis.current_weighted_apy)}</span>
              <span className="stat-label">APY</span>
            </div>
            <div className="stat-item">
              <span className="stat-value positive">+{fmt(analysis.yearly_earnings)}</span>
              <span className="stat-label">Yearly</span>
            </div>
            <div className="stat-item">
              <span className="stat-value">{fmt(totalSaved)}</span>
              <span className="stat-label">By Rules</span>
            </div>
          </div>
        )}

        {/* Opportunity Alert */}
        {analysis?.has_opportunity && analysis.best_vault_for_user && (
          <div className="opportunity-banner">
            <div className="opp-content">
              <span className="opp-label">Opportunity</span>
              <span className="opp-text">
                {fmt(analysis.idle_cash)} idle → {analysis.best_vault_for_user.name} ({pct(analysis.best_vault_for_user.apy)})
              </span>
            </div>
            <div className="opp-right">
              <span className="opp-gain">+{fmt(analysis.projected_gain_yearly)}/yr</span>
              <button className="opp-btn" onClick={() => navigate('/chat', { state: { autoMessage: 'Optimize my savings' } })}>
                Optimize
              </button>
            </div>
          </div>
        )}

        {/* Emergency Fund Ladder */}
        {emergencyFund?.exists && emergencyFund.stages && (
          <div className="section-card full-width">
            <div className="section-header">
              <h3>Emergency Fund Ladder</h3>
              <span className="ef-current">{fmt(emergencyFund.current)}</span>
            </div>
            <div className="ef-ladder">
              {emergencyFund.stages.map((stage) => (
                <div key={stage.stage} className={`ef-stage ${stage.complete ? 'complete' : ''}`}>
                  <div className="ef-stage-header">
                    <span className={`ef-stage-badge ${stage.complete ? 'complete' : ''}`}>
                      {stage.complete ? '✓' : stage.stage}
                    </span>
                    <div className="ef-stage-info">
                      <span className="ef-stage-name">{stage.name}</span>
                      <span className="ef-stage-target">{fmt(stage.target)}</span>
                    </div>
                  </div>
                  <div className="ef-stage-progress">
                    <div className="progress-bar">
                      <div 
                        className={`progress-fill ${stage.complete ? 'complete' : ''}`}
                        style={{ width: `${Math.min(stage.progress, 100)}%` }}
                      />
                    </div>
                    <span className="ef-stage-pct">{stage.progress.toFixed(0)}%</span>
                  </div>
                  <p className="ef-stage-desc">{stage.description}</p>
                  {stage.stage === 1 && !stage.complete && (
                    <div className="ef-unlock-msg">
                      <span>🔒</span> {fmt(stage.target - emergencyFund.current)} more to unlock investing
                    </div>
                  )}
                  {stage.stage === 1 && stage.complete && (
                    <div className="ef-unlocked-msg">
                      <span>🔓</span> Investing unlocked!
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Savings Rules */}
        <div className="section-card full-width">
          <div className="section-header">
            <h3>Savings Rules</h3>
            <button className="add-rule-btn" onClick={() => setShowCreateForm(true)}>
              + Add Rule
            </button>
          </div>

          {rules.length === 0 && !showCreateForm ? (
            <div className="empty-rules">
              <p>No automated savings rules yet.</p>
              <div className="rule-suggestions">
                <button onClick={() => { setFormData({...formData, name: 'Payday Sweep', trigger: 'income_detected', action: 'transfer', amount: '200'}); setShowCreateForm(true); }}>
                  💰 Payday Sweep
                </button>
                <button onClick={() => { setFormData({...formData, name: 'Round-ups', trigger: 'purchase', action: 'round_and_save'}); setShowCreateForm(true); }}>
                  🪙 Round-ups
                </button>
                <button onClick={() => { setFormData({...formData, name: 'Weekly Save', trigger: 'weekly', action: 'transfer', amount: '50'}); setShowCreateForm(true); }}>
                  📅 Weekly Save
                </button>
              </div>
            </div>
          ) : (
            <div className="rules-list">
              {rules.map((rule) => (
                <div key={rule.id} className={`rule-card ${rule.active ? 'active' : 'inactive'}`}>
                  <div className="rule-header">
                    <span className="rule-name">{rule.name}</span>
                    <span className={`rule-status ${rule.active ? 'active' : ''}`}>
                      {rule.active ? 'Active' : 'Paused'}
                    </span>
                  </div>
                  <div className="rule-details">
                    <span className="rule-trigger">{triggerLabels[rule.trigger] || rule.trigger}</span>
                    <span className="rule-action">
                      {rule.action === 'transfer' && fmt(rule.amount)}
                      {rule.action === 'transfer_percentage' && `${(rule.percentage * 100).toFixed(0)}%`}
                      {rule.action === 'round_and_save' && 'Round up'}
                    </span>
                    <span className="rule-dest">→ {destinationLabels[rule.destination] || rule.destination}</span>
                  </div>
                  <div className="rule-stats">
                    <span className="rule-saved">Saved: {fmt(rule.total_saved)}</span>
                    <span className="rule-runs">{rule.run_count} times</span>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Create Rule Form */}
          {showCreateForm && (
            <div className="create-rule-form">
              <h4>Create Savings Rule</h4>
              
              <div className="form-row">
                <label>Rule Name</label>
                <input
                  type="text"
                  placeholder="e.g., Payday Sweep"
                  value={formData.name}
                  onChange={(e) => setFormData({...formData, name: e.target.value})}
                />
              </div>

              <div className="form-row-group">
                <div className="form-row">
                  <label>Trigger</label>
                  <select value={formData.trigger} onChange={(e) => setFormData({...formData, trigger: e.target.value})}>
                    <option value="income_detected">After Payday</option>
                    <option value="purchase">Every Purchase</option>
                    <option value="budget_under">Under Budget</option>
                    <option value="weekly">Weekly</option>
                    <option value="monthly">Monthly</option>
                  </select>
                </div>

                <div className="form-row">
                  <label>Action</label>
                  <select value={formData.action} onChange={(e) => setFormData({...formData, action: e.target.value})}>
                    <option value="transfer">Transfer Fixed Amount</option>
                    <option value="transfer_percentage">Transfer Percentage</option>
                    <option value="round_and_save">Round Up Purchases</option>
                  </select>
                </div>
              </div>

              {formData.action === 'transfer' && (
                <div className="form-row">
                  <label>Amount</label>
                  <input
                    type="number"
                    placeholder="100"
                    value={formData.amount}
                    onChange={(e) => setFormData({...formData, amount: e.target.value})}
                  />
                </div>
              )}

              {formData.action === 'transfer_percentage' && (
                <div className="form-row">
                  <label>Percentage</label>
                  <input
                    type="number"
                    placeholder="10"
                    value={formData.percentage}
                    onChange={(e) => setFormData({...formData, percentage: e.target.value})}
                  />
                </div>
              )}

              <div className="form-row">
                <label>Destination</label>
                <select value={formData.destination} onChange={(e) => setFormData({...formData, destination: e.target.value})}>
                  <option value="savings_emergency">Emergency Fund</option>
                  <option value="savings_goals">Goals</option>
                  <option value="investment">Investment</option>
                </select>
              </div>

              <div className="form-actions">
                <button className="cancel-btn" onClick={() => setShowCreateForm(false)}>Cancel</button>
                <button className="create-btn" onClick={createRule} disabled={creating}>
                  {creating ? 'Creating...' : 'Create Rule'}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
