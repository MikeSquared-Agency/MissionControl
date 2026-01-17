import { useState, useRef, useEffect } from 'react'

interface KingInputProps {
  onSend: (content: string) => void
  disabled?: boolean
  placeholder?: string
}

export function KingInput({ onSend, disabled = false, placeholder = 'Tell the King what to build...' }: KingInputProps) {
  const [value, setValue] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current
    if (textarea) {
      textarea.style.height = 'auto'
      textarea.style.height = `${Math.min(textarea.scrollHeight, 120)}px`
    }
  }, [value])

  const handleSubmit = () => {
    const trimmed = value.trim()
    if (!trimmed || disabled) return

    onSend(trimmed)
    setValue('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="px-4 py-3 border-t border-amber-500/20 bg-amber-500/5">
      <div className="flex items-end gap-3">
        {/* Crown icon */}
        <div className="flex-shrink-0 pb-2">
          <span className="text-lg">👑</span>
        </div>

        {/* Input */}
        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={disabled}
            rows={1}
            className="w-full px-3 py-2 text-sm bg-gray-800/80 border border-amber-500/30 rounded-lg text-gray-100 placeholder-gray-500 resize-none focus:outline-none focus:border-amber-500/50 disabled:opacity-50 disabled:cursor-not-allowed"
            data-king-input
          />
        </div>

        {/* Send button */}
        <button
          onClick={handleSubmit}
          disabled={disabled || !value.trim()}
          className="flex-shrink-0 px-4 py-2 text-sm font-medium text-gray-900 bg-amber-500 hover:bg-amber-400 disabled:bg-amber-500/50 disabled:cursor-not-allowed rounded-lg transition-colors"
        >
          Command
        </button>
      </div>

      {/* Hint */}
      <p className="mt-2 text-[10px] text-amber-500/60 ml-9">
        Press Enter to send · Shift+Enter for new line
      </p>
    </div>
  )
}
