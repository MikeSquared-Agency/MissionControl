import { useStore, useAgentsNeedingAttention } from '../stores/useStore'
import type { Agent } from '../types'

interface AttentionBarProps {
  onRespond: (agentId: string, response: string) => void
}

export function AttentionBar({ onRespond }: AttentionBarProps) {
  const agentsNeedingAttention = useAgentsNeedingAttention()
  const selectAgent = useStore((s) => s.selectAgent)

  if (agentsNeedingAttention.length === 0) return null

  // Group by attention type
  const questions = agentsNeedingAttention.filter((a) => a.attention?.type === 'question')
  const permissions = agentsNeedingAttention.filter((a) => a.attention?.type === 'permission')
  const errors = agentsNeedingAttention.filter((a) => a.attention?.type === 'error')
  const complete = agentsNeedingAttention.filter((a) => a.attention?.type === 'complete')

  const handleAgentClick = (agent: Agent) => {
    selectAgent(agent.id)
  }

  return (
    <div className="bg-amber-500/10 border-b border-amber-500/20">
      <div className="px-4 py-2 flex items-center gap-4">
        {/* Icon and count */}
        <div className="flex items-center gap-2">
          <span className="text-lg animate-pulse">⚠️</span>
          <span className="text-xs font-medium text-amber-400">
            {agentsNeedingAttention.length} agent{agentsNeedingAttention.length !== 1 ? 's' : ''} need{agentsNeedingAttention.length === 1 ? 's' : ''} attention
          </span>
        </div>

        {/* Separator */}
        <div className="h-4 w-px bg-amber-500/30" />

        {/* Agent pills */}
        <div className="flex-1 flex items-center gap-2 overflow-x-auto">
          {/* Questions */}
          {questions.map((agent) => (
            <AttentionPill
              key={agent.id}
              agent={agent}
              icon="❓"
              color="amber"
              onClick={() => handleAgentClick(agent)}
              onRespond={onRespond}
            />
          ))}

          {/* Permissions */}
          {permissions.map((agent) => (
            <AttentionPill
              key={agent.id}
              agent={agent}
              icon="🔐"
              color="amber"
              onClick={() => handleAgentClick(agent)}
              onRespond={onRespond}
            />
          ))}

          {/* Errors */}
          {errors.map((agent) => (
            <AttentionPill
              key={agent.id}
              agent={agent}
              icon="❌"
              color="red"
              onClick={() => handleAgentClick(agent)}
              onRespond={onRespond}
            />
          ))}

          {/* Complete */}
          {complete.map((agent) => (
            <AttentionPill
              key={agent.id}
              agent={agent}
              icon="✅"
              color="green"
              onClick={() => handleAgentClick(agent)}
              onRespond={onRespond}
            />
          ))}
        </div>
      </div>
    </div>
  )
}

interface AttentionPillProps {
  agent: Agent
  icon: string
  color: 'amber' | 'red' | 'green'
  onClick: () => void
  onRespond: (agentId: string, response: string) => void
}

function AttentionPill({ agent, icon, color, onClick, onRespond }: AttentionPillProps) {
  const colorStyles = {
    amber: 'bg-amber-500/20 border-amber-500/30 text-amber-300 hover:bg-amber-500/30',
    red: 'bg-red-500/20 border-red-500/30 text-red-300 hover:bg-red-500/30',
    green: 'bg-green-500/20 border-green-500/30 text-green-300 hover:bg-green-500/30'
  }

  const attention = agent.attention

  return (
    <div className={`flex items-center gap-1.5 px-2 py-1 rounded border ${colorStyles[color]} transition-colors`}>
      {/* Click to select agent */}
      <button
        onClick={onClick}
        className="flex items-center gap-1.5"
        title={attention?.message}
      >
        <span className="text-sm">{icon}</span>
        <span className="text-[11px] font-medium truncate max-w-[120px]">
          {agent.name}
        </span>
      </button>

      {/* Quick response buttons */}
      {attention?.type === 'permission' && (
        <div className="flex items-center gap-1 ml-1 pl-1 border-l border-current/20">
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRespond(agent.id, 'Allow')
            }}
            className="px-1.5 py-0.5 text-[10px] font-medium text-green-400 hover:bg-green-500/20 rounded transition-colors"
          >
            Allow
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRespond(agent.id, 'Deny')
            }}
            className="px-1.5 py-0.5 text-[10px] font-medium text-red-400 hover:bg-red-500/20 rounded transition-colors"
          >
            Deny
          </button>
        </div>
      )}

      {attention?.type === 'question' && (
        <div className="flex items-center gap-1 ml-1 pl-1 border-l border-current/20">
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRespond(agent.id, 'Yes')
            }}
            className="px-1.5 py-0.5 text-[10px] font-medium text-blue-400 hover:bg-blue-500/20 rounded transition-colors"
          >
            Yes
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRespond(agent.id, 'No')
            }}
            className="px-1.5 py-0.5 text-[10px] font-medium text-gray-400 hover:bg-gray-500/20 rounded transition-colors"
          >
            No
          </button>
        </div>
      )}

      {attention?.type === 'error' && attention.retryable && (
        <div className="ml-1 pl-1 border-l border-current/20">
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRespond(agent.id, 'retry')
            }}
            className="px-1.5 py-0.5 text-[10px] font-medium text-amber-400 hover:bg-amber-500/20 rounded transition-colors"
          >
            Retry
          </button>
        </div>
      )}

      {attention?.type === 'complete' && (
        <div className="ml-1 pl-1 border-l border-current/20">
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRespond(agent.id, 'dismiss')
            }}
            className="px-1.5 py-0.5 text-[10px] font-medium text-gray-400 hover:bg-gray-500/20 rounded transition-colors"
          >
            Dismiss
          </button>
        </div>
      )}
    </div>
  )
}
