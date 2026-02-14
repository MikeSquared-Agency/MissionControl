import { useSwarmOverview } from '../../stores/useSwarmStore'
import type { WarrenAgent } from '../../types/swarm'

export function AgentGrid() {
  const overview = useSwarmOverview()
  const agents = overview?.warren?.agents ?? []

  return (
    <div className="rounded-lg bg-gray-800/50 border border-gray-800 p-3">
      <h3 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-3">
        Agent Fleet
        {agents.length > 0 && (
          <span className="ml-2 text-gray-600">({agents.length})</span>
        )}
      </h3>

      {agents.length === 0 ? (
        <div className="text-xs text-gray-600 text-center py-6">
          No agents connected
        </div>
      ) : (
        <div className="space-y-1.5 max-h-60 overflow-y-auto">
          {agents.map((agent) => (
            <AgentCard key={agent.id} agent={agent} />
          ))}
        </div>
      )}
    </div>
  )
}

function AgentCard({ agent }: { agent: WarrenAgent }) {
  const stateColor = getStateColor(agent.state)

  return (
    <div className="flex items-center gap-2 p-2 rounded bg-gray-900/50 hover:bg-gray-900/80 transition-colors">
      {/* State indicator */}
      <span className={`w-2 h-2 rounded-full flex-shrink-0 ${stateColor}`} />

      {/* Agent info */}
      <div className="flex-1 min-w-0">
        <div className="text-xs font-medium text-gray-300 truncate">
          {agent.name || agent.id}
        </div>
        <div className="flex items-center gap-2 text-[10px] text-gray-600">
          <span>{agent.state}</span>
          {agent.policy && <span>{agent.policy}</span>}
          {agent.connections !== undefined && (
            <span>{agent.connections} conn</span>
          )}
        </div>
      </div>
    </div>
  )
}

function getStateColor(state: string): string {
  switch (state) {
    case 'ready':
      return 'bg-green-500'
    case 'sleeping':
      return 'bg-yellow-500'
    case 'starting':
      return 'bg-blue-500 animate-pulse'
    case 'stopping':
      return 'bg-orange-500'
    default:
      return 'bg-gray-500'
  }
}
