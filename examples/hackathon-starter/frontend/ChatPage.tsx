import React, { useState, useEffect, useRef, useCallback } from 'react'
import { useLocation } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import './ChatPage.css'

interface Message {
  id: string
  role: 'user' | 'assistant'
  content: string
}

interface Conversation {
  id: string
  title: string
  updated_at: string
}

interface ConfirmDialog {
  show: boolean
  actionId: string
  summary: string
  content: string
}

interface ChatPageProps {
  wsUrl: string
  apiUrl: string
}

export function ChatPage({ wsUrl, apiUrl }: ChatPageProps) {
  const location = useLocation()
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [currentConversation, setCurrentConversation] = useState<string | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [isConnected, setIsConnected] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [streamingContent, setStreamingContent] = useState('')
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [confirmDialog, setConfirmDialog] = useState<ConfirmDialog>({
    show: false,
    actionId: '',
    summary: '',
    content: ''
  })
  const [autoMessageSent, setAutoMessageSent] = useState(false)
  
  const wsRef = useRef<WebSocket | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamingContent])

  // Handle auto-message from navigation (e.g., closing position from Trading page)
  useEffect(() => {
    const state = location.state as { autoMessage?: string } | null
    if (state?.autoMessage && isConnected && !autoMessageSent) {
      setAutoMessageSent(true)
      // Small delay to ensure WebSocket is fully ready
      setTimeout(() => {
        sendMessage(state.autoMessage)
      }, 500)
      // Clear the state so it doesn't re-trigger
      window.history.replaceState({}, document.title)
    }
  }, [location.state, isConnected, autoMessageSent])

  const fetchConversations = useCallback(async () => {
    try {
      const res = await fetch(`${apiUrl}/api/conversations`)
      const data = await res.json()
      setConversations(data.conversations || [])
    } catch (err) {
      console.error('Failed to fetch conversations:', err)
    }
  }, [apiUrl])

  useEffect(() => {
    fetchConversations()
  }, [fetchConversations])

  useEffect(() => {
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
          fetchConversations()
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
          fetchConversations()
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
          // Show custom confirmation dialog instead of window.confirm
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
  }, [wsUrl, fetchConversations])

  const startNewConversation = () => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ type: 'new_conversation' }))
      setMessages([])
      setCurrentConversation(null)
    }
  }

  const resumeConversation = (convId: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      setMessages([])
      wsRef.current.send(JSON.stringify({ 
        type: 'resume_conversation', 
        conversationId: convId 
      }))
    }
  }

  const deleteConversation = async (convId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    if (!confirm('Delete this conversation?')) return
    try {
      await fetch(`${apiUrl}/api/conversations?id=${convId}`, { method: 'DELETE' })
      fetchConversations()
      if (currentConversation === convId) {
        setCurrentConversation(null)
        setMessages([])
      }
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }

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

  const quickActions = [
    "What's my financial health score?",
    "Am I spending too much?",
    "Help me create a budget",
    "Show my portfolio status",
  ]

  return (
    <div className="chat-container">
      {/* Confirmation Dialog */}
      {confirmDialog.show && (
        <div className="confirm-overlay" onClick={handleCancel}>
          <div className="confirm-dialog" onClick={e => e.stopPropagation()}>
            <div className="confirm-icon">⚡</div>
            <h3 className="confirm-title">{confirmDialog.summary}</h3>
            <div className="confirm-content">
              <ReactMarkdown>{confirmDialog.content}</ReactMarkdown>
            </div>
            <div className="confirm-buttons">
              <button className="confirm-btn-cancel" onClick={handleCancel}>
                Cancel
              </button>
              <button className="confirm-btn-confirm" onClick={handleConfirm}>
                Confirm Trade
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Sidebar */}
      <aside className={`chat-sidebar ${sidebarOpen ? '' : 'collapsed'}`}>
        <div className="sidebar-top">
          <button className="new-chat-btn" onClick={startNewConversation}>
            + New Chat
          </button>
        </div>
        
        <div className="conversation-list">
          {conversations.map(conv => (
            <div
              key={conv.id}
              className={`conv-item ${currentConversation === conv.id ? 'active' : ''}`}
              onClick={() => resumeConversation(conv.id)}
            >
              <span className="conv-title">
                {conv.title || `Chat ${conv.id.slice(0, 8)}`}
              </span>
              <button 
                className="conv-delete"
                onClick={(e) => deleteConversation(conv.id, e)}
              >
                x
              </button>
            </div>
          ))}
        </div>

        <button 
          className="sidebar-toggle"
          onClick={() => setSidebarOpen(!sidebarOpen)}
        >
          {sidebarOpen ? '<<' : '>>'}
        </button>
      </aside>

      {/* Main */}
      <main className="chat-main">
        <header className="chat-header">
          <button className="menu-btn" onClick={() => setSidebarOpen(!sidebarOpen)}>
            Menu
          </button>
          <h1>Nim</h1>
          <span className={`status ${isConnected ? 'online' : 'offline'}`}>
            {isConnected ? 'Connected' : 'Offline'}
          </span>
        </header>

        <div className="messages">
          {messages.length === 0 && !streamingContent ? (
            <div className="welcome">
              <h2>How can I help you today?</h2>
              <div className="quick-actions">
                {quickActions.map((action, i) => (
                  <button key={i} onClick={() => sendMessage(action)}>
                    {action}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <>
              {messages.map(msg => (
                <div key={msg.id} className={`message ${msg.role}`}>
                  <div className="message-content">
                    {msg.role === 'assistant' ? (
                      <ReactMarkdown>{msg.content}</ReactMarkdown>
                    ) : (
                      <p>{msg.content}</p>
                    )}
                  </div>
                </div>
              ))}
              {streamingContent && (
                <div className="message assistant">
                  <div className="message-content">
                    <ReactMarkdown>{streamingContent}</ReactMarkdown>
                  </div>
                </div>
              )}
              {isLoading && !streamingContent && (
                <div className="message assistant">
                  <div className="message-content">
                    <span className="typing">Thinking...</span>
                  </div>
                </div>
              )}
            </>
          )}
          <div ref={messagesEndRef} />
        </div>

        <div className="input-area">
          <textarea
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            disabled={!isConnected || isLoading}
            rows={1}
          />
          <button 
            onClick={() => sendMessage()}
            disabled={!isConnected || isLoading || !inputValue.trim()}
          >
            Send
          </button>
        </div>
      </main>
    </div>
  )
}
