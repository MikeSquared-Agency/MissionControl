import { useState, useRef } from 'react'
import type { Agent } from '../types'
import { useStore, killAgent } from '../stores/useStore'
import { ContextMenu } from './ContextMenu'

interface AgentCardProps {
  agent: Agent
  selected: boolean
  onSelect: () => void
  onMove?: () => void
  onShare?: () => void
}

export function AgentCard({ agent, selected, onSelect, onMove, onShare }: AgentCardProps) {
  const personas = useStore((s) => s.personas)
  const [menuOpen, setMenuOpen] = useState(false)
  const [menuPosition, setMenuPosition] = useState({ x: 0, y: 0 })
  const menuButtonRef = useRef<HTMLButtonElement>(null)
  const persona = personas.find((p) => p.id === agent.persona)

  const statusColors: Record<string, string> = {
    starting: 'bg-yellow-500',
    working: 'bg-green-500',
    idle: 'bg-blue-500',
    waiting: 'bg-amber-500',
    error: 'bg-red-500',
    stopped: 'bg-gray-500'
  }

  const hasAttention = agent.attention !== null

  const handleKill = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (confirm('Kill this agent?')) {
      try {
        await killAgent(agent.id)
      } catch (err) {
        console.error('Failed to kill agent:', err)
      }
    }
  }

  return (
    <div
      onClick={onSelect}
      className={`
        mx-2 mb-1 px-2 py-2 rounded cursor-pointer transition-all group
        ${selected
          ? 'bg-gray-800 ring-1 ring-blue-500/50'
          : hasAttention
            ? 'bg-gray-850 ring-1 ring-amber-500/30 hover:bg-gray-800'
            : 'hover:bg-gray-800/50'
        }
      `}
    >
      {/* Top row: status, name, type badge */}
      <div className="flex items-center gap-2">
        {/* Status dot */}
        <span
          className={`
            w-2 h-2 rounded-full flex-shrink-0
            ${statusColors[agent.status]}
            ${(agent.status === 'working' || hasAttention) ? 'animate-pulse' : ''}
          `}
        />

        {/* Agent name */}
        <span className="text-xs font-medium text-gray-200 truncate flex-1">
          {agent.name}
        </span>

        {/* Persona badge */}
        {persona && (
          <span
            className="px-1.5 py-0.5 text-[10px] font-medium rounded"
            style={{
              backgroundColor: `${persona.color}20`,
              color: persona.color
            }}
          >
            {persona.name.slice(0, 2).toUpperCase()}
          </span>
        )}

        {/* Type badge */}
        <span className="px-1.5 py-0.5 text-[10px] font-mono text-gray-500 bg-gray-800 rounded">
          {agent.type === 'claude-code' ? 'CC' : 'PY'}
        </span>

        {/* Menu button */}
        <button
          ref={menuButtonRef}
          onClick={(e) => {
            e.stopPropagation()
            const rect = e.currentTarget.getBoundingClientRect()
            setMenuPosition({ x: rect.right, y: rect.bottom })
            setMenuOpen(true)
          }}
          className="p-0.5 text-gray-600 hover:text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity"
          aria-label="Agent menu"
        >
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="6" r="2" />
            <circle cx="12" cy="12" r="2" />
            <circle cx="12" cy="18" r="2" />
          </svg>
        </button>

        {/* Agent context menu */}
        <ContextMenu
          open={menuOpen}
          onClose={() => setMenuOpen(false)}
          position={menuPosition}
          items={[
            {
              label: 'Move to Zone',
              onClick: () => {
                setMenuOpen(false)
                onMove?.()
              }
            },
            {
              label: 'Share Context',
              onClick: () => {
                setMenuOpen(false)
                onShare?.()
              },
              disabled: agent.findings.length === 0
            },
            { divider: true },
            {
              label: 'Kill Agent',
              onClick: async () => {
                setMenuOpen(false)
                if (confirm('Kill this agent?')) {
                  try {
                    await killAgent(agent.id)
                  } catch (err) {
                    console.error('Failed to kill agent:', err)
                  }
                }
              },
              danger: true,
              disabled: agent.status === 'stopped'
            }
          ]}
        />
      </div>

      {/* Task (truncated) */}
      <div className="mt-1 text-[11px] text-gray-500 line-clamp-1">
        {agent.task}
      </div>

      {/* Bottom row: stats and attention */}
      <div className="mt-1.5 flex items-center gap-2">
        {/* Token count */}
        <span className="text-[10px] font-mono text-gray-600">
          {formatTokens(agent.tokens)}
        </span>

        {/* Cost */}
        {agent.cost > 0 && (
          <span className="text-[10px] font-mono text-gray-600">
            ${agent.cost.toFixed(2)}
          </span>
        )}

        {/* Attention indicator */}
        {hasAttention && (
          <div className="flex-1 flex items-center gap-1 justify-end">
            <span className="text-amber-500">
              {agent.attention?.type === 'question' && '❓'}
              {agent.attention?.type === 'permission' && '🔐'}
              {agent.attention?.type === 'error' && '❌'}
              {agent.attention?.type === 'complete' && '✅'}
            </span>
            <span className="text-[10px] text-amber-400 truncate max-w-[80px]">
              {agent.attention?.message}
            </span>
            <span className="text-[10px] text-amber-500/70">
              {formatTimeSince(agent.attention?.since || 0)}
            </span>
          </div>
        )}
      </div>

      {/* Kill button (only when selected and not stopped) */}
      {selected && agent.status !== 'stopped' && (
        <button
          onClick={handleKill}
          className="absolute top-2 right-2 p-1 text-gray-500 hover:text-red-400 transition-colors"
          title="Kill agent"
        >
          <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  )
}

function formatTokens(tokens: number): string {
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(1)}M`
  }
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}k`
  }
  return String(tokens)
}

function formatTimeSince(timestamp: number): string {
  if (!timestamp) return ''
  const seconds = Math.floor((Date.now() - timestamp) / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  return `${hours}h`
}
