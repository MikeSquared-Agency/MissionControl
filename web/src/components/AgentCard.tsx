import type { Agent } from '../types'
import { killAgent } from '../stores/agentStore'

interface AgentCardProps {
  agent: Agent
  selected: boolean
  onSelect: () => void
}

export function AgentCard({ agent, selected, onSelect }: AgentCardProps) {
  const statusColors: Record<string, string> = {
    starting: 'bg-yellow-500',
    running: 'bg-green-500',
    idle: 'bg-blue-500',
    error: 'bg-red-500',
    stopped: 'bg-gray-500'
  }

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
      className={`p-4 rounded-lg cursor-pointer transition-all ${
        selected
          ? 'bg-gray-700 ring-2 ring-blue-500'
          : 'bg-gray-800 hover:bg-gray-750'
      }`}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <span className={`w-3 h-3 rounded-full ${statusColors[agent.status]}`} />
          <div>
            <div className="font-medium text-white">{agent.id}</div>
            <div className="text-sm text-gray-400">{agent.type}</div>
          </div>
        </div>

        {agent.status !== 'stopped' && (
          <button
            onClick={handleKill}
            className="text-gray-500 hover:text-red-400 text-sm"
            title="Kill agent"
          >
            ✕
          </button>
        )}
      </div>

      <div className="mt-3 text-sm text-gray-300 line-clamp-2">
        {agent.task}
      </div>

      <div className="mt-3 flex items-center gap-4 text-xs text-gray-500">
        <span>{agent.status}</span>
        <span>{agent.tokens.toLocaleString()} tokens</span>
        {agent.error && (
          <span className="text-red-400" title={agent.error}>error</span>
        )}
      </div>
    </div>
  )
}
