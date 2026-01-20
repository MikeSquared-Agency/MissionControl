import { useState, useEffect } from 'react'

interface TypingIndicatorProps {
  text: string
  delay?: number
  onComplete?: () => void
}

export function TypingIndicator({ text, delay = 300, onComplete }: TypingIndicatorProps) {
  const [showDots, setShowDots] = useState(true)

  useEffect(() => {
    const timer = setTimeout(() => {
      setShowDots(false)
      onComplete?.()
    }, delay)

    return () => clearTimeout(timer)
  }, [delay, onComplete])

  if (showDots) {
    return (
      <div className="flex items-center gap-1 text-gray-400 h-6">
        <span
          className="w-2 h-2 bg-gray-500 rounded-full animate-bounce"
          style={{ animationDelay: '0ms' }}
        />
        <span
          className="w-2 h-2 bg-gray-500 rounded-full animate-bounce"
          style={{ animationDelay: '150ms' }}
        />
        <span
          className="w-2 h-2 bg-gray-500 rounded-full animate-bounce"
          style={{ animationDelay: '300ms' }}
        />
      </div>
    )
  }

  return <span className="text-gray-100">{text}</span>
}
