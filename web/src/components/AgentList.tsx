import { useAgentStore } from '../stores/agentStore'
import { AgentCard } from './AgentCard'

export function AgentList() {
  const agents = useAgentStore((s) => s.agents)
  const selectedAgentId = useAgentStore((s) => s.selectedAgentId)
  const selectAgent = useAgentStore((s) => s.selectAgent)

  if (agents.length === 0) {
    return (
      <div className="p-6 text-center text-gray-500">
        <p>No agents yet.</p>
        <p className="text-sm mt-2">Click "+ New Agent" to spawn one.</p>
      </div>
    )
  }

  return (
    <div className="p-4 space-y-3">
      {agents.map((agent) => (
        <AgentCard
          key={agent.id}
          agent={agent}
          selected={selectedAgentId === agent.id}
          onSelect={() => selectAgent(agent.id)}
        />
      ))}
    </div>
  )
}
