import { useEffect, useRef } from 'react'
import type { ConversationMessage as MessageType } from '../types'
import { ConversationMessage } from './ConversationMessage'
import { Findings } from './Findings'

interface ConversationViewProps {
  messages: MessageType[]
  findings: string[]
  onToggleToolCollapse?: (toolCallId: string) => void
  onQuickResponse?: (response: string) => void
}

export function ConversationView({
  messages,
  findings,
  onToggleToolCollapse,
  onQuickResponse
}: ConversationViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length])

  // Empty state
  if (messages.length === 0 && findings.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-4">
        <div className="text-center">
          <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-gray-800 flex items-center justify-center">
            <svg className="w-6 h-6 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z" />
            </svg>
          </div>
          <p className="text-sm text-gray-500">No messages yet</p>
          <p className="text-xs text-gray-600 mt-1">This agent was just spawned</p>
        </div>
      </div>
    )
  }

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto">
      <div className="p-4 space-y-4">
        {/* Messages */}
        {messages.map((msg, i) => (
          <ConversationMessage
            key={`${msg.timestamp}-${i}`}
            message={msg}
            onToggleToolCollapse={onToggleToolCollapse}
            onQuickResponse={onQuickResponse}
          />
        ))}

        {/* Findings */}
        <Findings findings={findings} />

        {/* Scroll anchor */}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
