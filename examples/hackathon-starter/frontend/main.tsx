import React, { useState, useEffect } from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Routes, Route, Link, useLocation } from 'react-router-dom'
// Apple HIG Design System - must be first
import './apple-hig-design-system.css'
import './styles.css'
import './TradingDashboard.css'
import './LandingPage.css'
import { TradingPage } from './TradingPage'
import './TradingPage.css'
import { ChatPage } from './ChatPage'
import './ChatPage.css'
import { BudgetPage } from './BudgetPage'
import './BudgetPage.css'
import { SubscriptionsPage } from './SubscriptionsPage'
import './SubscriptionsPage.css'
import { SmartSavingsPage } from './SmartSavingsPage'
import './SmartSavingsPage.css'
import { ToastProvider, useToast, setToastHandler } from './Toast'
import { ChatWidget } from './ChatWidget'
import './ChatWidget.css'

// Floating Chat Widget (hidden on /chat page)
function FloatingChat({ wsUrl, apiUrl }: { wsUrl: string; apiUrl: string }) {
  const location = useLocation()
  
  // Hide on the full chat page
  if (location.pathname === '/chat') return null
  
  return <ChatWidget wsUrl={wsUrl} apiUrl={apiUrl} />
}

// Auth state management
interface AuthState {
  token: string | null
  isAuthenticated: boolean
}

// Login Component
function LoginPage({ onLogin }: { onLogin: (token: string) => void }) {
  const [email, setEmail] = useState('')
  const [otp, setOtp] = useState('')
  const [step, setStep] = useState<'email' | 'otp'>('email')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081'

  const handleRequestOTP = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch(`${apiUrl}/auth/v1/otp`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          email: email.toLowerCase().trim(),
          type: 1  // OTP_TYPE_LOGIN
        })
      })
      
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || 'Failed to send OTP')
      }
      
      setStep('otp')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send OTP')
    } finally {
      setLoading(false)
    }
  }

  const handleVerifyOTP = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch(`${apiUrl}/auth/v1/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          email: email.toLowerCase().trim(),
          code: otp,
          type: 1  // OTP_TYPE_LOGIN
        })
      })
      
      if (!response.ok) {
        const data = await response.json()
        throw new Error(data.message || 'Invalid OTP')
      }
      
      const data = await response.json()
      // Store token and notify parent (API returns accessToken, not access_token)
      const accessToken = data.accessToken || data.access_token
      const refreshToken = data.refreshToken || data.refresh_token
      localStorage.setItem('liminal_token', accessToken)
      localStorage.setItem('liminal_refresh_token', refreshToken)
      onLogin(accessToken)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to verify OTP')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>Login to Liminal</h1>
        <p className="login-subtitle">Connect your Liminal account to use Nim</p>
        
        {error && <div className="login-error">{error}</div>}
        
        {step === 'email' ? (
          <form onSubmit={handleRequestOTP}>
            <div className="form-group">
              <label htmlFor="email">Email Address</label>
              <input
                id="email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
                required
                disabled={loading}
              />
            </div>
            <button type="submit" className="login-button" disabled={loading}>
              {loading ? 'Sending...' : 'Send OTP'}
            </button>
          </form>
        ) : (
          <form onSubmit={handleVerifyOTP}>
            <div className="form-group">
              <label htmlFor="otp">Enter OTP sent to {email}</label>
              <input
                id="otp"
                type="text"
                value={otp}
                onChange={(e) => setOtp(e.target.value)}
                placeholder="123456"
                required
                disabled={loading}
                autoFocus
              />
            </div>
            <button type="submit" className="login-button" disabled={loading}>
              {loading ? 'Verifying...' : 'Verify OTP'}
            </button>
            <button 
              type="button" 
              className="login-link"
              onClick={() => { setStep('email'); setOtp(''); setError(null); }}
            >
              Use different email
            </button>
          </form>
        )}
      </div>
    </div>
  )
}

// Journey status interface
interface JourneyStatus {
  step_1_chat: boolean
  step_2_budget: boolean
  step_3_subscriptions: boolean
  step_4_savings: boolean
  step_5_investing: boolean
  step_6_autopilot: boolean
  next_step: string
  next_step_number: number
  emergency_fund?: {
    current_stage: number
    stage_1_progress: number
  }
}

// Journey Progress Component
function JourneyProgress() {
  const [status, setStatus] = useState<JourneyStatus | null>(null)
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081'

  useEffect(() => {
    fetch(`${apiUrl}/api/journey-status`)
      .then(res => res.json())
      .then(data => setStatus(data))
      .catch(err => console.error('Failed to fetch journey status:', err))
  }, [apiUrl])

  if (!status) return null

  const steps = [
    { key: 'step_1_chat', label: 'Chat', icon: '💬', path: '/chat' },
    { key: 'step_2_budget', label: 'Budget', icon: '📊', path: '/budget' },
    { key: 'step_3_subscriptions', label: 'Subscriptions', icon: '🔄', path: '/subscriptions' },
    { key: 'step_4_savings', label: 'Savings', icon: '💰', path: '/savings' },
    { key: 'step_5_investing', label: 'Investing', icon: '📈', path: '/trading' },
    { key: 'step_6_autopilot', label: 'Autopilot', icon: '🤖', path: '/chat' },
  ]

  const completedCount = steps.filter(s => status[s.key as keyof JourneyStatus]).length

  return (
    <div className="journey-progress">
      <div className="journey-header">
        <h3>Your Journey</h3>
        <span className="journey-count">{completedCount}/6 steps</span>
      </div>
      <div className="journey-steps">
        {steps.map((step, index) => {
          const isComplete = status[step.key as keyof JourneyStatus] === true
          const isCurrent = status.next_step_number === index + 1
          const isLocked = step.key === 'step_5_investing' && !status.step_5_investing
          
          return (
            <Link 
              key={step.key}
              to={step.path}
              className={`journey-step ${isComplete ? 'complete' : ''} ${isCurrent ? 'current' : ''} ${isLocked ? 'locked' : ''}`}
            >
              <span className="step-icon">{isLocked ? '🔒' : step.icon}</span>
              <span className="step-label">{step.label}</span>
              {isComplete && <span className="step-check">✓</span>}
            </Link>
          )
        })}
      </div>
      {status.next_step && (
        <div className="journey-next">
          <span className="next-label">Next:</span>
          <span className="next-action">{status.next_step}</span>
        </div>
      )}
    </div>
  )
}

// Home Page Component
function HomePage() {
  return (
    <div className="landing-page">
      {/* Hero Section */}
      <section className="landing-hero">
        <div className="hero-content">
          <span className="hero-badge">AI-Powered Finance</span>
          <h1 className="hero-title">
            Your Money,<br />
            <span className="highlight">Multiplied by AI</span>
          </h1>
          <p className="hero-subtitle">
            Autonomous agents that optimize your spending, grow your savings, and manage your wealth.
          </p>
          <div className="hero-actions">
            <Link to="/chat" className="primary-button">
              Chat with Nim →
            </Link>
          </div>
        </div>
      </section>

      {/* Journey Progress */}
      <section className="journey-section">
        <JourneyProgress />
      </section>

      {/* Features Grid */}
      <section className="features-section">
        <div className="section-header">
          <h2 className="section-title">Tools</h2>
          <p className="section-desc">Everything you need to manage your finances</p>
        </div>
        <div className="features-grid">
          <Link to="/budget" className="feature-card feature-card-link">
            <span className="feature-icon">📊</span>
            <h3 className="feature-title">Budget Planner</h3>
            <p className="feature-desc">
              Track spending patterns and forecast your financial future.
            </p>
            <span className="feature-badge">Open</span>
          </Link>

          <Link to="/savings" className="feature-card feature-card-link">
            <span className="feature-icon">💰</span>
            <h3 className="feature-title">Smart Savings</h3>
            <p className="feature-desc">
              Automatically move cash to high-yield accounts.
            </p>
            <span className="feature-badge">Open</span>
          </Link>

          <Link to="/subscriptions" className="feature-card feature-card-link">
            <span className="feature-icon">🔄</span>
            <h3 className="feature-title">Subscriptions</h3>
            <p className="feature-desc">
              Find and cancel forgotten subscriptions.
            </p>
            <span className="feature-badge">Open</span>
          </Link>

          <Link to="/trading" className="feature-card feature-card-link">
            <span className="feature-icon">📈</span>
            <h3 className="feature-title">Trading</h3>
            <p className="feature-desc">
              Monitor positions and execute trades.
            </p>
            <span className="feature-badge">Open</span>
          </Link>
        </div>
      </section>
    </div>
  )
}

// Toast handler initializer component
function ToastHandlerInit() {
  const toastContext = useToast()
  useEffect(() => {
    setToastHandler(toastContext)
  }, [toastContext])
  return null
}

// Theme type
type Theme = 'light' | 'dark' | 'system'

// Main App with Router and Auth
function App() {
  const baseWsUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8081/ws'
  const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:8081'
  
  const [auth, setAuth] = useState<AuthState>(() => {
    const token = localStorage.getItem('liminal_token')
    return {
      token,
      isAuthenticated: !!token
    }
  })

  const [demoMode, setDemoMode] = useState<boolean>(true)
  const [demoLoading, setDemoLoading] = useState<boolean>(false)
  
  // Theme state
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('theme')
    return (saved as Theme) || 'system'
  })

  // Apply theme to document
  useEffect(() => {
    const root = document.documentElement
    
    const applyTheme = (resolvedTheme: 'light' | 'dark') => {
      root.setAttribute('data-theme', resolvedTheme)
      root.classList.remove('light', 'dark')
      root.classList.add(resolvedTheme)
    }
    
    if (theme === 'system') {
      // Detect system preference
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
      applyTheme(isDark ? 'dark' : 'light')
      
      // Listen for system theme changes
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      const handler = (e: MediaQueryListEvent) => applyTheme(e.matches ? 'dark' : 'light')
      mediaQuery.addEventListener('change', handler)
      return () => mediaQuery.removeEventListener('change', handler)
    } else {
      applyTheme(theme)
    }
    
    localStorage.setItem('theme', theme)
  }, [theme])

  const cycleTheme = () => {
    setTheme(current => {
      if (current === 'system') return 'light'
      if (current === 'light') return 'dark'
      return 'system'
    })
  }

  const getThemeIcon = () => {
    if (theme === 'light') return '☀️'
    if (theme === 'dark') return '🌙'
    return '💻'
  }

  const getThemeLabel = () => {
    if (theme === 'light') return 'Light'
    if (theme === 'dark') return 'Dark'
    return 'Auto'
  }

  // Fetch initial demo mode state
  useEffect(() => {
    fetch(`${apiUrl}/api/demo-mode`)
      .then(res => res.json())
      .then(data => setDemoMode(data.demo_mode))
      .catch(err => console.error('Failed to fetch demo mode:', err))
  }, [apiUrl])

  const toggleDemoMode = async () => {
    setDemoLoading(true)
    try {
      const res = await fetch(`${apiUrl}/api/demo-mode`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enable: !demoMode })
      })
      const data = await res.json()
      setDemoMode(data.demo_mode)
    } catch (err) {
      console.error('Failed to toggle demo mode:', err)
    } finally {
      setDemoLoading(false)
    }
  }

  const handleLogin = (token: string) => {
    setAuth({ token, isAuthenticated: true })
  }

  const handleLogout = () => {
    localStorage.removeItem('liminal_token')
    localStorage.removeItem('liminal_refresh_token')
    setAuth({ token: null, isAuthenticated: false })
  }

  // Build WebSocket URL with token
  const wsUrl = auth.token ? `${baseWsUrl}?token=${auth.token}` : baseWsUrl

  // If not authenticated, show login
  if (!auth.isAuthenticated) {
    return (
      <ToastProvider>
        <ToastHandlerInit />
        <LoginPage onLogin={handleLogin} />
      </ToastProvider>
    )
  }

  return (
    <ToastProvider>
      <ToastHandlerInit />
      <BrowserRouter>
      {/* Top bar controls */}
      <div style={{
        position: 'fixed',
        top: '12px',
        right: '12px',
        zIndex: 1000,
        display: 'flex',
        gap: '8px',
        alignItems: 'center'
      }}>
        {/* Theme Toggle */}
        <button
          onClick={cycleTheme}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            padding: '8px 14px',
            background: 'var(--fill-tertiary)',
            border: 'none',
            borderRadius: '20px',
            cursor: 'pointer',
            backdropFilter: 'blur(10px)',
            fontSize: '13px',
            color: 'var(--label-primary)',
            fontWeight: 500,
          }}
          title={`Theme: ${getThemeLabel()}`}
        >
          <span>{getThemeIcon()}</span>
          <span>{getThemeLabel()}</span>
        </button>

        {/* Demo Mode Toggle */}
        <button
          onClick={toggleDemoMode}
          disabled={demoLoading}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            padding: '8px 14px',
            background: 'var(--fill-tertiary)',
            border: 'none',
            borderRadius: '20px',
            cursor: demoLoading ? 'wait' : 'pointer',
            backdropFilter: 'blur(10px)',
            fontSize: '13px',
            color: demoMode ? 'var(--system-orange)' : 'var(--label-primary)',
            fontWeight: 500,
          }}
        >
          <span>{demoMode ? '🎮' : '🔗'}</span>
          <span>{demoMode ? 'Demo' : 'Live'}</span>
        </button>

        {/* Sign Out */}
        <button 
          onClick={handleLogout}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            padding: '8px 14px',
            background: 'var(--fill-tertiary)',
            border: 'none',
            borderRadius: '20px',
            cursor: 'pointer',
            backdropFilter: 'blur(10px)',
            fontSize: '13px',
            color: 'var(--label-primary)',
            fontWeight: 500,
          }}
        >
          Sign Out
        </button>
      </div>
      
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/chat" element={<ChatPage wsUrl={wsUrl} apiUrl={apiUrl} />} />
        <Route path="/trading" element={<TradingPage />} />
        <Route path="/budget" element={<BudgetPage />} />
        <Route path="/subscriptions" element={<SubscriptionsPage />} />
        <Route path="/savings" element={<SmartSavingsPage />} />
      </Routes>

      {/* Floating chat widget on non-chat pages */}
      <FloatingChat wsUrl={wsUrl} apiUrl={apiUrl} />
    </BrowserRouter>
    </ToastProvider>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
