import React, { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import './ChatWidget.css'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
}

interface ConfirmDialog {
  show: boolean
  actionId: string
  summary: string
  content: string
}

interface ChatWidgetProps {
  wsUrl: string
  apiUrl: string
}

export function ChatWidget({ wsUrl, apiUrl: _apiUrl }: ChatWidgetProps) {
  // apiUrl reserved for future use (e.g., fetching conversations)
  void _apiUrl
  const navigate = useNavigate()
  const [isOpen, setIsOpen] = useState(false)
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [isConnected, setIsConnected] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [streamingContent, setStreamingContent] = useState('')
  const [currentConversation, setCurrentConversation] = useState<string | null>(null)
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialog>({
    show: false,
    actionId: '',
    summary: '',
    content: ''
  })
  
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  // Connect WebSocket when widget opens
  useEffect(() => {
    if (!isOpen) return
    
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
    }

    ws.onclose = () => {
      setIsConnected(false)
    }

    ws.onmessage = (event) => {
      const data = JSON.parse(event.data)

      switch (data.type) {
        case 'conversation_started':
          setCurrentConversation(data.conversationId)
          // Don't clear messages - preserve the user's first message that was added optimistically
          break

        case 'conversation_resumed':
          setCurrentConversation(data.conversationId)
          if (data.messages) {
            setMessages(data.messages.map((m: any, i: number) => ({
              id: m.id || `msg-${i}`,
              role: m.role,
              content: m.content,
            })))
          } else {
            setMessages([])
          }
          break

        case 'text_chunk':
          setStreamingContent(prev => prev + data.content)
          break

        case 'text':
          if (data.content) {
            setMessages(prev => [...prev, {
              id: `msg-${Date.now()}`,
              role: 'assistant',
              content: data.content,
            }])
          }
          setStreamingContent('')
          break

        case 'complete':
          setIsLoading(false)
          setStreamingContent('')
          break

        case 'error':
          console.error('Server error:', data.content)
          setIsLoading(false)
          setStreamingContent('')
          setMessages(prev => [...prev, {
            id: `error-${Date.now()}`,
            role: 'assistant',
            content: `**Error:** ${data.content}`,
          }])
          break

        case 'confirm_request':
          setConfirmDialog({
            show: true,
            actionId: data.actionId,
            summary: data.summary || 'Confirm Action',
            content: data.content || ''
          })
          break
      }
    }

    return () => ws.close()
  }, [wsUrl, isOpen])

  const sendMessage = (text?: string) => {
    const content = text || inputValue.trim()
    if (!content || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return

    if (!currentConversation) {
      wsRef.current.send(JSON.stringify({ type: 'new_conversation' }))
      setTimeout(() => {
        wsRef.current?.send(JSON.stringify({ type: 'message', content }))
      }, 100)
    } else {
      wsRef.current.send(JSON.stringify({ type: 'message', content }))
    }

    setMessages(prev => [...prev, {
      id: `msg-${Date.now()}`,
      role: 'user',
      content,
    }])
    setInputValue('')
    setIsLoading(true)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  const handleConfirm = () => {
    if (wsRef.current?.readyState === WebSocket.OPEN && confirmDialog.actionId) {
      wsRef.current.send(JSON.stringify({ type: 'confirm', actionId: confirmDialog.actionId }))
      setIsLoading(true)
    }
    setConfirmDialog({ show: false, actionId: '', summary: '', content: '' })
  }

  const handleCancel = () => {
    if (wsRef.current?.readyState === WebSocket.OPEN && confirmDialog.actionId) {
      wsRef.current.send(JSON.stringify({ type: 'cancel', actionId: confirmDialog.actionId }))
    }
    setIsLoading(false)
    setConfirmDialog({ show: false, actionId: '', summary: '', content: '' })
  }

  const handleExpand = () => {
    // Navigate to full chat page with current conversation
    navigate('/chat', { state: { conversationId: currentConversation } })
  }

  const quickActions = [
    "What's my journey status?",
    "Help me get started",
    "Show my budget",
  ]

  return (
    <>
      {/* Floating Chat Button */}
      <button
        className={`chat-widget-fab ${isOpen ? 'open' : ''}`}
        onClick={() => setIsOpen(!isOpen)}
        title={isOpen ? 'Close chat' : 'Chat with Nim'}
      >
        {isOpen ? '✕' : '💬'}
      </button>

      {/* Chat Widget Panel */}
      {isOpen && (
        <div className="chat-widget-panel">
          {/* Header */}
          <div className="chat-widget-header">
            <div className="chat-widget-header-left">
              <div className="chat-widget-avatar">N</div>
              <div className="chat-widget-title">
                <span className="chat-widget-name">Nim</span>
                <span className={`chat-widget-status ${isConnected ? 'connected' : ''}`}>
                  {isConnected ? 'Online' : 'Connecting...'}
                </span>
              </div>
            </div>
            <div className="chat-widget-header-actions">
              <button 
                className="chat-widget-expand-btn"
                onClick={handleExpand}
                title="Open full chat"
                aria-label="Open full chat"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M15 3h6v6M9 21H3v-6M21 3l-7 7M3 21l7-7"/>
                </svg>
              </button>
              <button 
                className="chat-widget-close-btn"
                onClick={() => setIsOpen(false)}
                title="Minimize"
                aria-label="Minimize chat"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M18 6L6 18M6 6l12 12"/>
                </svg>
              </button>
            </div>
          </div>

          {/* Messages */}
          <div className="chat-widget-messages">
            {messages.length === 0 && !streamingContent && (
              <div className="chat-widget-welcome">
                <div className="chat-widget-welcome-icon">👋</div>
                <h3>Hi! I'm Nim</h3>
                <p>Your AI financial assistant. How can I help you today?</p>
                <div className="chat-widget-quick-actions">
                  {quickActions.map((action, i) => (
                    <button
                      key={i}
                      className="chat-widget-quick-btn"
                      onClick={() => sendMessage(action)}
                      disabled={!isConnected}
                    >
                      {action}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {messages.map((msg) => (
              <div key={msg.id} className={`chat-widget-message ${msg.role}`}>
                <div className="chat-widget-message-content">
                  {msg.role === 'assistant' ? (
                    <ReactMarkdown>{msg.content}</ReactMarkdown>
                  ) : (
                    msg.content
                  )}
                </div>
              </div>
            ))}

            {streamingContent && (
              <div className="chat-widget-message assistant">
                <div className="chat-widget-message-content">
                  <ReactMarkdown>{streamingContent}</ReactMarkdown>
                </div>
              </div>
            )}

            {isLoading && !streamingContent && (
              <div className="chat-widget-message assistant">
                <div className="chat-widget-message-content">
                  <div className="chat-widget-typing">
                    <span></span><span></span><span></span>
                  </div>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Confirmation Dialog */}
          {confirmDialog.show && (
            <div className="chat-widget-confirm">
              <div className="chat-widget-confirm-content">
                <h4>{confirmDialog.summary}</h4>
                <div className="chat-widget-confirm-text">
                  <ReactMarkdown>{confirmDialog.content}</ReactMarkdown>
                </div>
                <div className="chat-widget-confirm-actions">
                  <button className="confirm-btn cancel" onClick={handleCancel}>
                    Cancel
                  </button>
                  <button className="confirm-btn approve" onClick={handleConfirm}>
                    Confirm
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Input */}
          <div className="chat-widget-input-area">
            <textarea
              className="chat-widget-input"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={isConnected ? "Ask Nim anything..." : "Connecting..."}
              disabled={!isConnected || isLoading}
              rows={1}
            />
            <button
              className="chat-widget-send-btn"
              onClick={() => sendMessage()}
              disabled={!isConnected || !inputValue.trim() || isLoading}
            >
              ↑
            </button>
          </div>
        </div>
      )}
    </>
  )
}
