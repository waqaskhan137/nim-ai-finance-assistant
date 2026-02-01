import React, { createContext, useContext, useState, useCallback, useEffect } from 'react'

// Toast types
export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface Toast {
  id: string
  type: ToastType
  title: string
  message?: string
  duration?: number
}

interface ToastContextType {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => void
  removeToast: (id: string) => void
  success: (title: string, message?: string) => void
  error: (title: string, message?: string) => void
  warning: (title: string, message?: string) => void
  info: (title: string, message?: string) => void
}

const ToastContext = createContext<ToastContextType | null>(null)

// Icons for each toast type
const toastIcons: Record<ToastType, string> = {
  success: '\u2713', // Checkmark
  error: '\u2715',   // X mark
  warning: '\u26A0', // Warning triangle
  info: '\u2139',    // Info circle
}

// Single toast component
function ToastItem({ toast, onClose }: { toast: Toast; onClose: () => void }) {
  const [isExiting, setIsExiting] = useState(false)

  useEffect(() => {
    const duration = toast.duration || 4000
    const timer = setTimeout(() => {
      setIsExiting(true)
      setTimeout(onClose, 250) // Wait for exit animation
    }, duration)

    return () => clearTimeout(timer)
  }, [toast.duration, onClose])

  const handleClose = () => {
    setIsExiting(true)
    setTimeout(onClose, 250)
  }

  return (
    <div className={`toast ${toast.type} ${isExiting ? 'toast-exit' : ''}`}>
      <span className="toast-icon">{toastIcons[toast.type]}</span>
      <div className="toast-content">
        <span className="toast-title">{toast.title}</span>
        {toast.message && <span className="toast-message">{toast.message}</span>}
      </div>
      <button className="toast-close" onClick={handleClose} aria-label="Close">
        {'\u2715'}
      </button>
    </div>
  )
}

// Toast container component
function ToastContainer({ toasts, removeToast }: { toasts: Toast[]; removeToast: (id: string) => void }) {
  if (toasts.length === 0) return null

  return (
    <div className="toast-container">
      {toasts.map((toast) => (
        <ToastItem 
          key={toast.id} 
          toast={toast} 
          onClose={() => removeToast(toast.id)} 
        />
      ))}
    </div>
  )
}

// Toast Provider
export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = useCallback((toast: Omit<Toast, 'id'>) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
    setToasts((prev) => [...prev, { ...toast, id }])
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const success = useCallback((title: string, message?: string) => {
    addToast({ type: 'success', title, message })
  }, [addToast])

  const error = useCallback((title: string, message?: string) => {
    addToast({ type: 'error', title, message, duration: 6000 }) // Errors stay longer
  }, [addToast])

  const warning = useCallback((title: string, message?: string) => {
    addToast({ type: 'warning', title, message, duration: 5000 })
  }, [addToast])

  const info = useCallback((title: string, message?: string) => {
    addToast({ type: 'info', title, message })
  }, [addToast])

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast, success, error, warning, info }}>
      {children}
      <ToastContainer toasts={toasts} removeToast={removeToast} />
    </ToastContext.Provider>
  )
}

// Hook to use toasts
export function useToast() {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider')
  }
  return context
}

// Standalone toast function (for use outside React components)
let toastHandler: ToastContextType | null = null

export function setToastHandler(handler: ToastContextType) {
  toastHandler = handler
}

export const toast = {
  success: (title: string, message?: string) => toastHandler?.success(title, message),
  error: (title: string, message?: string) => toastHandler?.error(title, message),
  warning: (title: string, message?: string) => toastHandler?.warning(title, message),
  info: (title: string, message?: string) => toastHandler?.info(title, message),
}
