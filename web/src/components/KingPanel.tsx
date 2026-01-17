import { useRef, useEffect } from 'react'
import { useStore, sendKingMessage } from '../stores/useStore'
import { KingHeader } from './KingHeader'
import { TeamOverview } from './TeamOverview'
import { KingInput } from './KingInput'
import { KingAction } from './KingAction'
import type { KingMessage } from '../types'

interface KingPanelProps {
  onExit: () => void
  onAgentClick?: (agentId: string) => void
}

export function KingPanel({ onExit, onAgentClick }: KingPanelProps) {
  const kingConversation = useStore((s) => s.kingConversation)
  const addKingMessage = useStore((s) => s.addKingMessage)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [kingConversation])

  const handleSend = async (content: string) => {
    // Add user message immediately
    const userMessage: KingMessage = {
      role: 'user',
      content,
      timestamp: Date.now()
    }
    addKingMessage(userMessage)

    // Send to backend
    try {
      await sendKingMessage(content)
    } catch (err) {
      console.error('Failed to send king message:', err)
      // Add error message
      addKingMessage({
        role: 'assistant',
        content: `Error: ${err instanceof Error ? err.message : 'Failed to send message'}`,
        timestamp: Date.now()
      })
    }
  }

  return (
    <div className="flex-1 flex flex-col bg-gray-950">
      {/* Header */}
      <KingHeader onExit={onExit} />

      {/* Team overview */}
      <TeamOverview onAgentClick={onAgentClick} />

      {/* Conversation */}
      <div className="flex-1 overflow-y-auto px-4 py-4">
        {kingConversation.length === 0 ? (
          <div className="h-full flex items-center justify-center">
            <div className="text-center max-w-md">
              <span className="text-4xl">👑</span>
              <h3 className="mt-4 text-lg font-medium text-amber-400">
                Welcome to King Mode
              </h3>
              <p className="mt-2 text-sm text-gray-500">
                Tell me what you want to build. I'll create and manage a team of AI agents to accomplish your goal.
              </p>
              <div className="mt-6 space-y-2 text-[11px] text-gray-600 text-left">
                <p className="flex items-center gap-2">
                  <span className="text-green-500">•</span>
                  "Build a REST API for user authentication"
                </p>
                <p className="flex items-center gap-2">
                  <span className="text-blue-500">•</span>
                  "Refactor the payment module and add tests"
                </p>
                <p className="flex items-center gap-2">
                  <span className="text-purple-500">•</span>
                  "Review the codebase for security issues"
                </p>
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {kingConversation.map((message, index) => (
              <KingMessageBubble key={index} message={message} />
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}
      </div>

      {/* Input */}
      <KingInput onSend={handleSend} />
    </div>
  )
}

interface KingMessageBubbleProps {
  message: KingMessage
}

function KingMessageBubble({ message }: KingMessageBubbleProps) {
  const isUser = message.role === 'user'

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[80%] ${
          isUser
            ? 'bg-amber-500/20 border border-amber-500/30 rounded-lg rounded-br-sm'
            : 'bg-gray-800/80 border border-gray-700/50 rounded-lg rounded-bl-sm'
        }`}
      >
        {/* Thinking (collapsed by default) */}
        {message.thinking && (
          <details className="px-4 pt-3">
            <summary className="text-[10px] text-gray-500 cursor-pointer hover:text-gray-400">
              View thinking...
            </summary>
            <div className="mt-2 text-xs text-gray-500 italic whitespace-pre-wrap">
              {message.thinking}
            </div>
          </details>
        )}

        {/* Content */}
        <div className="px-4 py-3">
          <p className={`text-sm whitespace-pre-wrap ${isUser ? 'text-amber-100' : 'text-gray-200'}`}>
            {message.content}
          </p>
        </div>

        {/* Actions */}
        {message.actions && message.actions.length > 0 && (
          <div className="px-4 pb-3 space-y-2">
            {message.actions.map((action, index) => (
              <KingAction key={index} action={action} />
            ))}
          </div>
        )}

        {/* Timestamp */}
        <div className="px-4 pb-2">
          <span className="text-[10px] text-gray-600">
            {new Date(message.timestamp).toLocaleTimeString()}
          </span>
        </div>
      </div>
    </div>
  )
}
