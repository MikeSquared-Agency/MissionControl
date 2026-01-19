import { useRef, useEffect, useState } from 'react'
import { useStore } from '../stores/useStore'
import { useMissionStore, startKing, stopKing, sendKingMessage as sendMissionKingMessage, answerKingQuestion } from '../stores/useMissionStore'
import { KingHeader } from './KingHeader'
import { TeamOverview } from './TeamOverview'
import { KingInput } from './KingInput'
import { KingAction } from './KingAction'
import { WorkersPanel } from './WorkersPanel'
import type { KingMessage, KingQuestion } from '../types'

interface KingPanelProps {
  onExit: () => void
  onAgentClick?: (agentId: string) => void
}

export function KingPanel({ onExit, onAgentClick }: KingPanelProps) {
  const kingConversation = useStore((s) => s.kingConversation)
  const addKingMessage = useStore((s) => s.addKingMessage)
  const kingRunning = useMissionStore((s) => s.kingRunning)
  const kingQuestion = useMissionStore((s) => s.kingQuestion)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const [startingKing, setStartingKing] = useState(false)
  const [showWorkers, setShowWorkers] = useState(false)
  const [answeringQuestion, setAnsweringQuestion] = useState(false)

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [kingConversation])

  const handleStartKing = async () => {
    setStartingKing(true)
    try {
      await startKing()
    } catch (err) {
      console.error('Failed to start King:', err)
      addKingMessage({
        role: 'assistant',
        content: `Error starting King: ${err instanceof Error ? err.message : 'Unknown error'}`,
        timestamp: Date.now()
      })
    } finally {
      setStartingKing(false)
    }
  }

  const handleStopKing = async () => {
    try {
      await stopKing()
    } catch (err) {
      console.error('Failed to stop King:', err)
    }
  }

  const handleSend = async (content: string) => {
    // Start King if not running
    if (!kingRunning) {
      await handleStartKing()
    }

    // Add user message immediately
    const userMessage: KingMessage = {
      role: 'user',
      content,
      timestamp: Date.now()
    }
    addKingMessage(userMessage)

    // Send to backend
    try {
      await sendMissionKingMessage(content)
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

  const handleAnswerQuestion = async (optionIndex: number) => {
    if (!kingQuestion) return

    setAnsweringQuestion(true)
    try {
      // Add user's selection to conversation
      addKingMessage({
        role: 'user',
        content: `Selected: ${kingQuestion.options[optionIndex]}`,
        timestamp: Date.now()
      })
      await answerKingQuestion(optionIndex)
    } catch (err) {
      console.error('Failed to answer question:', err)
      addKingMessage({
        role: 'assistant',
        content: `Error: ${err instanceof Error ? err.message : 'Failed to answer'}`,
        timestamp: Date.now()
      })
    } finally {
      setAnsweringQuestion(false)
    }
  }

  return (
    <div className="flex-1 flex flex-col bg-gray-950 overflow-hidden">
      {/* Header with status */}
      <div className="flex items-center justify-between px-4 py-2 bg-gray-900 border-b border-gray-800">
        <div className="flex items-center gap-3">
          <KingHeader onExit={onExit} />
          {/* King status indicator */}
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${kingRunning ? 'bg-green-500 animate-pulse' : 'bg-gray-500'}`} />
            <span className="text-[10px] text-gray-500">
              {startingKing ? 'Starting...' : kingRunning ? 'Running' : 'Stopped'}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowWorkers(!showWorkers)}
            className={`px-2 py-1 text-[10px] rounded transition-colors ${showWorkers ? 'bg-gray-700 text-gray-200' : 'text-gray-500 hover:text-gray-400'}`}
          >
            Workers
          </button>
          {kingRunning && (
            <button
              onClick={handleStopKing}
              className="px-2 py-1 text-[10px] text-red-400 hover:text-red-300 transition-colors"
            >
              Stop
            </button>
          )}
        </div>
      </div>

      {/* Team overview */}
      <TeamOverview onAgentClick={onAgentClick} />

      {/* Workers panel (collapsible sidebar) */}
      {showWorkers && (
        <div className="absolute right-0 top-20 w-72 h-[calc(100%-10rem)] bg-gray-900 border-l border-gray-800 p-4 overflow-y-auto z-10">
          <WorkersPanel />
        </div>
      )}

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

      {/* Question Panel or Input */}
      {kingQuestion ? (
        <KingQuestionPanel
          question={kingQuestion}
          onAnswer={handleAnswerQuestion}
          disabled={answeringQuestion}
        />
      ) : (
        <KingInput onSend={handleSend} />
      )}
    </div>
  )
}

// Question Panel Component
interface KingQuestionPanelProps {
  question: KingQuestion
  onAnswer: (optionIndex: number) => void
  disabled?: boolean
}

function KingQuestionPanel({ question, onAnswer, disabled }: KingQuestionPanelProps) {
  const [selectedIndex, setSelectedIndex] = useState(question.selected)

  return (
    <div className="border-t border-gray-800 bg-gray-900/50 p-4">
      <div className="max-w-2xl mx-auto">
        {/* Question */}
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-amber-400">?</span>
            <span className="text-sm font-medium text-gray-200">Claude is asking:</span>
          </div>
          <p className="text-sm text-gray-300 ml-5">{question.question}</p>
        </div>

        {/* Options */}
        <div className="space-y-2 mb-4">
          {question.options.map((option, index) => (
            <button
              key={index}
              onClick={() => setSelectedIndex(index)}
              disabled={disabled}
              className={`w-full text-left px-4 py-3 rounded-lg border transition-all ${
                selectedIndex === index
                  ? 'border-amber-500/50 bg-amber-500/10 text-amber-100'
                  : 'border-gray-700 bg-gray-800/50 text-gray-300 hover:border-gray-600 hover:bg-gray-800'
              } ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
            >
              <div className="flex items-center gap-3">
                <div className={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                  selectedIndex === index ? 'border-amber-400' : 'border-gray-500'
                }`}>
                  {selectedIndex === index && (
                    <div className="w-2 h-2 rounded-full bg-amber-400" />
                  )}
                </div>
                <span className="text-sm">{option}</span>
              </div>
            </button>
          ))}
        </div>

        {/* Submit Button */}
        <button
          onClick={() => onAnswer(selectedIndex)}
          disabled={disabled}
          className={`w-full py-2 px-4 rounded-lg font-medium text-sm transition-all ${
            disabled
              ? 'bg-gray-700 text-gray-400 cursor-not-allowed'
              : 'bg-amber-500 text-gray-900 hover:bg-amber-400'
          }`}
        >
          {disabled ? 'Sending...' : 'Confirm Selection'}
        </button>
      </div>
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
