import { useAgentStore } from '../stores/agentStore'

interface HeaderProps {
  status: 'connecting' | 'connected' | 'disconnected' | 'error'
  onSpawnClick: () => void
}

export function Header({ status, onSpawnClick }: HeaderProps) {
  const agents = useAgentStore((s) => s.agents)

  const activeCount = agents.filter((a) => a.status === 'running').length
  const totalTokens = agents.reduce((sum, a) => sum + a.tokens, 0)

  const statusColor = {
    connecting: 'bg-yellow-500',
    connected: 'bg-green-500',
    disconnected: 'bg-gray-500',
    error: 'bg-red-500'
  }[status]

  return (
    <header className="bg-gray-800 border-b border-gray-700 px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <h1 className="text-xl font-semibold text-white">Agent Orchestra</h1>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <span className={`w-2 h-2 rounded-full ${statusColor}`} />
            <span>{status}</span>
          </div>
        </div>

        <div className="flex items-center gap-6">
          <div className="flex items-center gap-4 text-sm text-gray-400">
            <span>{agents.length} agents</span>
            <span>{activeCount} active</span>
            <span>{totalTokens.toLocaleString()} tokens</span>
          </div>

          <button
            onClick={onSpawnClick}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            + New Agent
          </button>
        </div>
      </div>
    </header>
  )
}
