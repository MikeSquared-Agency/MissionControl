import type { Agent } from '../types'
import { useStore, killAgent } from '../stores/useStore'

interface AgentHeaderProps {
  agent: Agent
}

export function AgentHeader({ agent }: AgentHeaderProps) {
  const personas = useStore((s) => s.personas)
  const persona = personas.find((p) => p.id === agent.persona)

  const statusColors: Record<string, { bg: string; text: string }> = {
    starting: { bg: 'bg-yellow-500/20', text: 'text-yellow-400' },
    working: { bg: 'bg-green-500/20', text: 'text-green-400' },
    idle: { bg: 'bg-blue-500/20', text: 'text-blue-400' },
    waiting: { bg: 'bg-amber-500/20', text: 'text-amber-400' },
    error: { bg: 'bg-red-500/20', text: 'text-red-400' },
    stopped: { bg: 'bg-gray-500/20', text: 'text-gray-400' }
  }

  const { bg, text } = statusColors[agent.status] || statusColors.idle

  const handleKill = async () => {
    if (confirm('Kill this agent? This cannot be undone.')) {
      try {
        await killAgent(agent.id)
      } catch (err) {
        console.error('Failed to kill agent:', err)
      }
    }
  }

  return (
    <div className="px-4 py-3 border-b border-gray-800/50 bg-gray-900">
      {/* Row 1: Name, status, persona, tokens, cost, kill */}
      <div className="flex items-center gap-3">
        {/* Status dot */}
        <span
          className={`w-2.5 h-2.5 rounded-full ${agent.status === 'working' ? 'animate-pulse' : ''}`}
          style={{
            backgroundColor: agent.status === 'working' ? '#22c55e'
              : agent.status === 'error' ? '#ef4444'
              : agent.status === 'waiting' ? '#f59e0b'
              : agent.status === 'stopped' ? '#6b7280'
              : '#3b82f6'
          }}
        />

        {/* Agent name */}
        <h2 className="text-sm font-medium text-gray-100">
          {agent.name}
        </h2>

        {/* Status badge */}
        <span className={`px-2 py-0.5 text-[10px] font-medium rounded ${bg} ${text}`}>
          {agent.status}
        </span>

        {/* Persona badge */}
        {persona && (
          <span
            className="px-2 py-0.5 text-[10px] font-medium rounded"
            style={{
              backgroundColor: `${persona.color}20`,
              color: persona.color
            }}
          >
            {persona.name}
          </span>
        )}

        {/* Spacer */}
        <div className="flex-1" />

        {/* Stats */}
        <div className="flex items-center gap-3 text-xs text-gray-500">
          <span className="font-mono">{formatTokens(agent.tokens)} tokens</span>
          {agent.cost > 0 && (
            <span className="font-mono">${agent.cost.toFixed(2)}</span>
          )}
        </div>

        {/* Kill button */}
        {agent.status !== 'stopped' && (
          <button
            onClick={handleKill}
            className="px-2 py-1 text-[11px] font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors"
          >
            Kill
          </button>
        )}
      </div>

      {/* Row 2: Task */}
      <div className="mt-2 text-xs text-gray-400">
        <span className="text-gray-600">Task:</span> {agent.task}
      </div>

      {/* Row 3: Working directory */}
      {agent.workingDir && (
        <div className="mt-1 text-[11px] font-mono text-gray-600">
          {agent.workingDir}
        </div>
      )}

      {/* Row 4: Available tools (from persona) */}
      {persona && persona.tools.length > 0 && (
        <div className="mt-2 flex items-center gap-1.5">
          <span className="text-[10px] text-gray-600">Tools:</span>
          <div className="flex flex-wrap gap-1">
            {persona.tools.map((tool) => (
              <span
                key={tool}
                className="px-1.5 py-0.5 text-[10px] font-mono text-gray-500 bg-gray-800 rounded"
              >
                {tool}
              </span>
            ))}
          </div>
        </div>
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
