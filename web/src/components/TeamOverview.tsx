import { useStore } from '../stores/useStore'

interface TeamOverviewProps {
  onAgentClick?: (agentId: string) => void
}

export function TeamOverview({ onAgentClick }: TeamOverviewProps) {
  const agents = useStore((s) => s.agents)
  const personas = useStore((s) => s.personas)
  const zones = useStore((s) => s.zones)

  // Group agents by zone
  const agentsByZone = zones.map((zone) => ({
    zone,
    agents: agents.filter((a) => a.zone === zone.id)
  })).filter((g) => g.agents.length > 0)

  // Unzoned agents (shouldn't happen but handle it)
  const unzonedAgents = agents.filter((a) => !zones.find((z) => z.id === a.zone))

  const statusColors: Record<string, string> = {
    starting: 'bg-yellow-500',
    working: 'bg-green-500',
    idle: 'bg-blue-500',
    waiting: 'bg-amber-500',
    error: 'bg-red-500',
    stopped: 'bg-gray-500'
  }

  if (agents.length === 0) {
    return (
      <div className="px-4 py-3 text-center text-xs text-gray-500">
        No agents active. Tell the King what to build.
      </div>
    )
  }

  return (
    <div className="px-4 py-3 border-b border-amber-500/20">
      {/* Stats summary */}
      <div className="flex items-center gap-4 mb-3 text-[11px]">
        <span className="text-gray-500">
          <span className="text-amber-400 font-mono">{agents.length}</span> agents
        </span>
        <span className="text-gray-500">
          <span className="text-green-400 font-mono">
            {agents.filter((a) => a.status === 'working').length}
          </span> working
        </span>
        {agents.filter((a) => a.status === 'waiting' || a.attention).length > 0 && (
          <span className="text-amber-500 animate-pulse">
            <span className="font-mono">
              {agents.filter((a) => a.status === 'waiting' || a.attention).length}
            </span> waiting
          </span>
        )}
      </div>

      {/* Agent badges by zone */}
      <div className="space-y-2">
        {agentsByZone.map(({ zone, agents: zoneAgents }) => (
          <div key={zone.id}>
            <div className="flex items-center gap-2 mb-1">
              <span
                className="w-2 h-2 rounded-full flex-shrink-0"
                style={{ backgroundColor: zone.color }}
              />
              <span className="text-[10px] text-gray-500">{zone.name}</span>
            </div>
            <div className="flex flex-wrap gap-1.5 ml-4">
              {zoneAgents.map((agent) => {
                const persona = personas.find((p) => p.id === agent.persona)
                return (
                  <button
                    key={agent.id}
                    onClick={() => onAgentClick?.(agent.id)}
                    className="group flex items-center gap-1.5 px-2 py-1 bg-gray-800/80 hover:bg-gray-700 border border-gray-700/50 rounded transition-colors"
                    title={`${agent.name} - ${agent.status}`}
                  >
                    {/* Status dot */}
                    <span
                      className={`w-1.5 h-1.5 rounded-full ${statusColors[agent.status]} ${
                        (agent.status === 'working' || agent.attention) ? 'animate-pulse' : ''
                      }`}
                    />

                    {/* Agent name */}
                    <span className="text-[11px] text-gray-300 group-hover:text-gray-100">
                      {agent.name}
                    </span>

                    {/* Persona badge */}
                    {persona && (
                      <span
                        className="text-[9px] font-medium px-1 rounded"
                        style={{
                          backgroundColor: `${persona.color}20`,
                          color: persona.color
                        }}
                      >
                        {persona.name.slice(0, 2).toUpperCase()}
                      </span>
                    )}

                    {/* Attention indicator */}
                    {agent.attention && (
                      <span className="text-amber-500 text-[10px]">
                        {agent.attention.type === 'question' && '❓'}
                        {agent.attention.type === 'permission' && '🔐'}
                        {agent.attention.type === 'error' && '❌'}
                      </span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>
        ))}

        {/* Unzoned agents */}
        {unzonedAgents.length > 0 && (
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className="w-2 h-2 rounded-full bg-gray-500 flex-shrink-0" />
              <span className="text-[10px] text-gray-500">Unassigned</span>
            </div>
            <div className="flex flex-wrap gap-1.5 ml-4">
              {unzonedAgents.map((agent) => (
                <button
                  key={agent.id}
                  onClick={() => onAgentClick?.(agent.id)}
                  className="flex items-center gap-1.5 px-2 py-1 bg-gray-800/80 hover:bg-gray-700 border border-gray-700/50 rounded transition-colors"
                >
                  <span
                    className={`w-1.5 h-1.5 rounded-full ${statusColors[agent.status]}`}
                  />
                  <span className="text-[11px] text-gray-300">{agent.name}</span>
                </button>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
