import { useState, useRef, useEffect } from 'react'

interface ChatInputProps {
  onSend: (message: string) => void
  disabled?: boolean
  placeholder?: string
}

export function ChatInput({ onSend, disabled, placeholder = 'Send a message...' }: ChatInputProps) {
  const [message, setMessage] = useState('')
  const [sending, setSending] = useState(false)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  // Auto-resize textarea
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.style.height = 'auto'
      inputRef.current.style.height = `${Math.min(inputRef.current.scrollHeight, 120)}px`
    }
  }, [message])

  const handleSubmit = async () => {
    const trimmed = message.trim()
    if (!trimmed || disabled || sending) return

    setSending(true)
    try {
      await onSend(trimmed)
      setMessage('')
    } finally {
      setSending(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="px-4 py-3 border-t border-gray-800/50 bg-gray-900">
      <div className="flex items-end gap-2">
        <textarea
          ref={inputRef}
          data-chat-input
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={disabled || sending}
          rows={1}
          className="flex-1 px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 resize-none focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
          style={{ minHeight: '38px', maxHeight: '120px' }}
        />
        <button
          onClick={handleSubmit}
          disabled={disabled || sending || !message.trim()}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors flex-shrink-0"
        >
          {sending ? (
            <span className="flex items-center gap-1.5">
              <svg className="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
              </svg>
              Sending
            </span>
          ) : (
            'Send'
          )}
        </button>
      </div>
      <div className="mt-1.5 text-[10px] text-gray-600">
        Press <kbd className="px-1 py-0.5 bg-gray-800 rounded font-mono">Enter</kbd> to send,{' '}
        <kbd className="px-1 py-0.5 bg-gray-800 rounded font-mono">Shift+Enter</kbd> for new line
      </div>
    </div>
  )
}
