import type { ConversationMessage as MessageType } from '../types'
import { ToolCallBlock } from './ToolCallBlock'

interface ConversationMessageProps {
  message: MessageType
  onToggleToolCollapse?: (toolCallId: string) => void
  onQuickResponse?: (response: string) => void
}

export function ConversationMessage({
  message,
  onToggleToolCollapse,
  onQuickResponse
}: ConversationMessageProps) {
  const { role, content, toolCalls, isQuestion, isPermission, timestamp } = message

  // Format timestamp
  const time = new Date(timestamp).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit'
  })

  // User message
  if (role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[80%]">
          <div className="px-3 py-2 bg-blue-600 text-white text-xs rounded-lg rounded-br-sm">
            {content}
          </div>
          <div className="text-[10px] text-gray-600 text-right mt-0.5">
            {time}
          </div>
        </div>
      </div>
    )
  }

  // Error message
  if (role === 'error') {
    return (
      <div className="px-3 py-2 bg-red-500/10 border border-red-500/20 rounded">
        <div className="flex items-start gap-2">
          <span className="text-red-500 mt-0.5">✕</span>
          <div className="flex-1">
            <p className="text-xs text-red-400">{content}</p>
          </div>
        </div>
        <div className="text-[10px] text-gray-600 mt-1">
          {time}
        </div>
      </div>
    )
  }

  // Assistant message
  return (
    <div className="max-w-[95%]">
      {/* Content text */}
      {content && (
        <div className="text-xs text-gray-300 whitespace-pre-wrap">
          {content}
        </div>
      )}

      {/* Tool calls */}
      {toolCalls && toolCalls.length > 0 && (
        <div className="space-y-1">
          {toolCalls.map((tc) => (
            <ToolCallBlock
              key={tc.id}
              toolCall={tc}
              onToggleCollapse={() => onToggleToolCollapse?.(tc.id)}
            />
          ))}
        </div>
      )}

      {/* Quick response buttons for questions/permissions */}
      {(isQuestion || isPermission) && (
        <div className="mt-3 flex items-center gap-2">
          {isPermission ? (
            <>
              <button
                onClick={() => onQuickResponse?.('Allow')}
                className="px-3 py-1.5 text-[11px] font-medium text-green-400 bg-green-500/10 hover:bg-green-500/20 border border-green-500/30 rounded transition-colors"
              >
                Allow
              </button>
              <button
                onClick={() => onQuickResponse?.('Deny')}
                className="px-3 py-1.5 text-[11px] font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 rounded transition-colors"
              >
                Deny
              </button>
            </>
          ) : (
            <>
              <button
                onClick={() => onQuickResponse?.('Yes')}
                className="px-3 py-1.5 text-[11px] font-medium text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded transition-colors"
              >
                Yes
              </button>
              <button
                onClick={() => onQuickResponse?.('No')}
                className="px-3 py-1.5 text-[11px] font-medium text-gray-400 bg-gray-500/10 hover:bg-gray-500/20 border border-gray-500/30 rounded transition-colors"
              >
                No
              </button>
            </>
          )}
        </div>
      )}

      {/* Timestamp */}
      <div className="text-[10px] text-gray-600 mt-1">
        {time}
      </div>
    </div>
  )
}
