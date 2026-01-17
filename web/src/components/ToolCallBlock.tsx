import { useState } from 'react'
import type { ToolCall } from '../types'

interface ToolCallBlockProps {
  toolCall: ToolCall
  onToggleCollapse?: () => void
}

export function ToolCallBlock({ toolCall, onToggleCollapse }: ToolCallBlockProps) {
  const [localCollapsed, setLocalCollapsed] = useState(toolCall.collapsed)
  const collapsed = toolCall.collapsed ?? localCollapsed

  const handleToggle = () => {
    setLocalCollapsed(!collapsed)
    onToggleCollapse?.()
  }

  // Format args for display
  const argsStr = JSON.stringify(toolCall.args, null, 2)
  const argsPreview = JSON.stringify(toolCall.args).slice(0, 80)

  return (
    <div className="mt-2 border border-gray-800 rounded overflow-hidden">
      {/* Header - clickable to expand/collapse */}
      <button
        onClick={handleToggle}
        className="w-full flex items-center gap-2 px-2 py-1.5 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        {/* Collapse arrow */}
        <svg
          className={`w-3 h-3 text-gray-500 transition-transform flex-shrink-0 ${collapsed ? '' : 'rotate-90'}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
        </svg>

        {/* Tool name */}
        <span className="text-[11px] font-mono font-medium text-blue-400">
          {toolCall.tool}
        </span>

        {/* Args preview (when collapsed) */}
        {collapsed && (
          <span className="text-[10px] font-mono text-gray-600 truncate flex-1">
            {argsPreview}
            {argsPreview.length >= 80 && '...'}
          </span>
        )}

        {/* Status indicator */}
        {toolCall.result !== undefined ? (
          <span className="text-[10px] text-green-500">✓</span>
        ) : (
          <span className="text-[10px] text-yellow-500 animate-pulse">●</span>
        )}
      </button>

      {/* Expanded content */}
      {!collapsed && (
        <div className="border-t border-gray-800">
          {/* Arguments */}
          <div className="px-2 py-1.5 bg-gray-900/50">
            <div className="text-[10px] text-gray-500 mb-1">Arguments</div>
            <pre className="text-[11px] font-mono text-gray-400 whitespace-pre-wrap overflow-x-auto max-h-32 overflow-y-auto">
              {argsStr}
            </pre>
          </div>

          {/* Result */}
          {toolCall.result !== undefined && (
            <div className="px-2 py-1.5 border-t border-gray-800 bg-gray-900/30">
              <div className="text-[10px] text-gray-500 mb-1">Result</div>
              <pre className="text-[11px] font-mono text-gray-400 whitespace-pre-wrap overflow-x-auto max-h-48 overflow-y-auto">
                {toolCall.result || '(empty)'}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
