import { useState, useEffect } from 'react'
import { useWebSocket } from './hooks/useWebSocket'
import { useAgentStore, fetchAgents } from './stores/agentStore'
import { Header } from './components/Header'
import { AgentList } from './components/AgentList'
import { EventLog } from './components/EventLog'
import { SpawnDialog } from './components/SpawnDialog'

function App() {
  const [spawnOpen, setSpawnOpen] = useState(false)
  const { status } = useWebSocket()
  const setAgents = useAgentStore((s) => s.setAgents)
  const selectedAgentId = useAgentStore((s) => s.selectedAgentId)
  const agents = useAgentStore((s) => s.agents)

  // Fetch agents on mount
  useEffect(() => {
    fetchAgents()
      .then(setAgents)
      .catch((err) => console.error('Failed to fetch agents:', err))
  }, [setAgents])

  const selectedAgent = agents.find((a) => a.id === selectedAgentId)

  return (
    <div className="min-h-screen flex flex-col">
      <Header status={status} onSpawnClick={() => setSpawnOpen(true)} />

      <div className="flex-1 flex">
        {/* Sidebar - Agent List */}
        <aside className="w-80 border-r border-gray-700 bg-gray-850 overflow-y-auto">
          <AgentList />
        </aside>

        {/* Main Content */}
        <main className="flex-1 flex flex-col">
          {selectedAgent ? (
            <>
              {/* Selected Agent Header */}
              <div className="px-6 py-4 border-b border-gray-700 bg-gray-800">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-lg font-medium text-white">
                      Agent {selectedAgent.id}
                    </h2>
                    <p className="text-sm text-gray-400 mt-1">
                      {selectedAgent.task}
                    </p>
                  </div>
                  <div className="text-sm text-gray-500">
                    {selectedAgent.status} · {selectedAgent.tokens.toLocaleString()} tokens
                  </div>
                </div>
              </div>

              {/* Event Log */}
              <div className="flex-1 overflow-y-auto bg-gray-900">
                <EventLog />
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-gray-500">
              <div className="text-center">
                <p className="text-lg">Select an agent to view details</p>
                <p className="text-sm mt-2">
                  Or click "+ New Agent" to spawn one
                </p>
              </div>
            </div>
          )}
        </main>
      </div>

      <SpawnDialog open={spawnOpen} onClose={() => setSpawnOpen(false)} />
    </div>
  )
}

export default App
