import { useSelectedAgent, useStore, sendMessage, respondToAttention } from '../stores/useStore'
import { AgentHeader } from './AgentHeader'
import { AttentionAlert } from './AttentionAlert'
import { ConversationView } from './ConversationView'
import { ChatInput } from './ChatInput'

export function AgentPanel() {
  const selectedAgent = useSelectedAgent()
  const toggleToolCallCollapse = useStore((s) => s.toggleToolCallCollapse)
  const addMessage = useStore((s) => s.addMessage)

  // Empty state - no agent selected
  if (!selectedAgent) {
    return (
      <div className="flex-1 flex items-center justify-center bg-gray-950">
        <div className="text-center">
          <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800/50 flex items-center justify-center">
            <svg className="w-8 h-8 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
            </svg>
          </div>
          <p className="text-sm text-gray-500">Select an agent to view details</p>
          <p className="text-xs text-gray-600 mt-2">
            Or press <kbd className="px-1.5 py-0.5 bg-gray-800 rounded text-gray-400 font-mono text-[10px]">⌘N</kbd> to spawn one
          </p>
        </div>
      </div>
    )
  }

  // Handle sending a message
  const handleSendMessage = async (content: string) => {
    // Add user message to conversation immediately
    addMessage(selectedAgent.id, {
      role: 'user',
      content,
      timestamp: Date.now()
    })

    // Send to backend
    try {
      await sendMessage(selectedAgent.id, content)
    } catch (err) {
      // Add error message to conversation
      addMessage(selectedAgent.id, {
        role: 'error',
        content: err instanceof Error ? err.message : 'Failed to send message',
        timestamp: Date.now()
      })
    }
  }

  // Handle quick response (from attention alert)
  const handleQuickResponse = async (response: string) => {
    try {
      await respondToAttention(selectedAgent.id, response)
    } catch (err) {
      console.error('Failed to respond:', err)
      // Fallback to sending as a message
      handleSendMessage(response)
    }
  }

  // Handle tool collapse toggle
  const handleToggleToolCollapse = (toolCallId: string) => {
    toggleToolCallCollapse(selectedAgent.id, toolCallId)
  }

  const isDisabled = selectedAgent.status === 'stopped'

  return (
    <div className="flex-1 flex flex-col bg-gray-950">
      {/* Agent header */}
      <AgentHeader agent={selectedAgent} />

      {/* Attention alert (if applicable) */}
      {selectedAgent.attention && (
        <AttentionAlert
          attention={selectedAgent.attention}
          onRespond={handleQuickResponse}
        />
      )}

      {/* Conversation view */}
      <ConversationView
        messages={selectedAgent.conversation}
        findings={selectedAgent.findings}
        onToggleToolCollapse={handleToggleToolCollapse}
        onQuickResponse={handleQuickResponse}
      />

      {/* Chat input */}
      <ChatInput
        onSend={handleSendMessage}
        disabled={isDisabled}
        placeholder={isDisabled ? 'Agent has stopped' : 'Send a message...'}
      />
    </div>
  )
}
