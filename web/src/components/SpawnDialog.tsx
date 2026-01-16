import { useState } from 'react'
import { spawnAgent } from '../stores/agentStore'
import type { AgentType } from '../types'

interface SpawnDialogProps {
  open: boolean
  onClose: () => void
}

export function SpawnDialog({ open, onClose }: SpawnDialogProps) {
  const [type, setType] = useState<AgentType>('python')
  const [task, setTask] = useState('')
  const [agent, setAgent] = useState('v1_basic')
  const [workdir, setWorkdir] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  if (!open) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!task.trim()) return

    setLoading(true)
    setError('')

    try {
      await spawnAgent({
        type,
        task: task.trim(),
        workdir: workdir.trim() || undefined,
        agent: type === 'python' ? agent : undefined
      })
      setTask('')
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to spawn agent')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
        <h2 className="text-xl font-semibold text-white mb-4">Spawn New Agent</h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Type</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setType('python')}
                className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition-colors ${
                  type === 'python'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                }`}
              >
                Python Agent
              </button>
              <button
                type="button"
                onClick={() => setType('claude')}
                className={`flex-1 py-2 px-4 rounded-lg text-sm font-medium transition-colors ${
                  type === 'claude'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                }`}
              >
                Claude Code
              </button>
            </div>
          </div>

          {type === 'python' && (
            <div>
              <label className="block text-sm text-gray-400 mb-1">Agent Version</label>
              <select
                value={agent}
                onChange={(e) => setAgent(e.target.value)}
                className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white"
              >
                <option value="v0_minimal">v0_minimal (bash only)</option>
                <option value="v1_basic">v1_basic (full tools)</option>
                <option value="v2_todo">v2_todo (with planning)</option>
                <option value="v3_subagent">v3_subagent (with delegation)</option>
              </select>
            </div>
          )}

          <div>
            <label className="block text-sm text-gray-400 mb-1">Task</label>
            <textarea
              value={task}
              onChange={(e) => setTask(e.target.value)}
              placeholder="Describe what the agent should do..."
              rows={3}
              className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white placeholder-gray-500 resize-none"
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">
              Working Directory <span className="text-gray-600">(optional)</span>
            </label>
            <input
              type="text"
              value={workdir}
              onChange={(e) => setWorkdir(e.target.value)}
              placeholder="/path/to/project"
              className="w-full bg-gray-700 border border-gray-600 rounded-lg px-3 py-2 text-white placeholder-gray-500"
            />
          </div>

          {error && (
            <div className="text-red-400 text-sm">{error}</div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2 px-4 bg-gray-700 hover:bg-gray-600 text-gray-300 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !task.trim()}
              className="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-800 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
            >
              {loading ? 'Spawning...' : 'Spawn'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
