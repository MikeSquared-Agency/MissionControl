import { useState, useEffect } from 'react'
import { useStore, spawnAgent } from '../stores/useStore'
import { toast } from '../stores/useToast'
import { Modal } from './Modal'
import { Spinner } from './Spinner'
import type { AgentType, Persona } from '../types'

interface SpawnDialogProps {
  open: boolean
  onClose: () => void
}

export function SpawnDialog({ open, onClose }: SpawnDialogProps) {
  const personas = useStore((s) => s.personas)
  const zones = useStore((s) => s.zones)
  const addAgent = useStore((s) => s.addAgent)
  const selectAgent = useStore((s) => s.selectAgent)

  const [selectedPersona, setSelectedPersona] = useState<string | null>(null)
  const [type, setType] = useState<AgentType>('claude-code')
  const [name, setName] = useState('')
  const [task, setTask] = useState('')
  const [zone, setZone] = useState('default')
  const [workingDir, setWorkingDir] = useState('')
  const [agent, setAgent] = useState('v1_basic')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setSelectedPersona(null)
      setType('claude-code')
      setName('')
      setTask('')
      setZone('default')
      setWorkingDir('')
      setAgent('v1_basic')
      setError('')
    }
  }, [open])

  // Update working dir when zone changes
  useEffect(() => {
    const selectedZone = zones.find((z) => z.id === zone)
    if (selectedZone?.workingDir) {
      setWorkingDir(selectedZone.workingDir)
    }
  }, [zone, zones])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!task.trim()) return

    setLoading(true)
    setError('')

    try {
      const agentName = name.trim() || `agent-${Date.now().toString(36)}`
      const newAgent = await spawnAgent({
        type,
        name: agentName,
        task: task.trim(),
        persona: selectedPersona || undefined,
        zone,
        workingDir: workingDir.trim() || undefined,
        agent: type === 'python' ? agent : undefined
      })
      addAgent(newAgent)
      selectAgent(newAgent.id)
      toast.success(`Agent "${agentName}" spawned successfully`)
      onClose()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to spawn agent'
      setError(errorMsg)
      toast.error(errorMsg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Spawn Agent" width="md">
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Persona selector - 2x2 grid */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-2">Persona</label>
          <div className="grid grid-cols-2 gap-2">
            {personas.map((persona) => (
              <PersonaCard
                key={persona.id}
                persona={persona}
                selected={selectedPersona === persona.id}
                onClick={() => setSelectedPersona(
                  selectedPersona === persona.id ? null : persona.id
                )}
              />
            ))}
          </div>
        </div>

        {/* Type selector */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-2">Type</label>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setType('claude-code')}
              className={`flex-1 py-2 text-xs font-medium rounded transition-colors ${
                type === 'claude-code'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
              }`}
            >
              Claude Code
            </button>
            <button
              type="button"
              onClick={() => setType('python')}
              className={`flex-1 py-2 text-xs font-medium rounded transition-colors ${
                type === 'python'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-800 text-gray-400 hover:bg-gray-700'
              }`}
            >
              Python Agent
            </button>
          </div>
        </div>

        {/* Python agent version (only for python type) */}
        {type === 'python' && (
          <div>
            <label className="block text-[11px] text-gray-500 mb-2">Agent Version</label>
            <select
              value={agent}
              onChange={(e) => setAgent(e.target.value)}
              className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
            >
              <option value="v0_minimal">v0_minimal (bash only)</option>
              <option value="v1_basic">v1_basic (full tools)</option>
              <option value="v2_todo">v2_todo (with planning)</option>
              <option value="v3_subagent">v3_subagent (with delegation)</option>
            </select>
          </div>
        )}

        {/* Name */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-2">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., code-reviewer-1"
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* Task */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-2">
            Task <span className="text-red-400">*</span>
          </label>
          <textarea
            value={task}
            onChange={(e) => setTask(e.target.value)}
            placeholder="Describe what the agent should do..."
            rows={3}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 resize-none focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* Zone */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-2">Zone</label>
          <select
            value={zone}
            onChange={(e) => setZone(e.target.value)}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
          >
            {zones.map((z) => (
              <option key={z.id} value={z.id}>
                {z.name}
              </option>
            ))}
          </select>
        </div>

        {/* Working directory */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-2">
            Working Directory
          </label>
          <input
            type="text"
            value={workingDir}
            onChange={(e) => setWorkingDir(e.target.value)}
            placeholder="/path/to/project"
            className="w-full px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* Error */}
        {error && (
          <div className="px-3 py-2 text-xs text-red-400 bg-red-500/10 border border-red-500/20 rounded">
            {error}
          </div>
        )}

        {/* Actions */}
        <div className="flex gap-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            className="flex-1 py-2 text-sm text-gray-400 bg-gray-800 hover:bg-gray-700 rounded transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={loading || !task.trim()}
            className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors flex items-center justify-center gap-2"
          >
            {loading && <Spinner size="sm" />}
            {loading ? 'Spawning...' : 'Spawn Agent'}
          </button>
        </div>

        {/* Hint */}
        <p className="text-[10px] text-gray-600 text-center">
          Press ⌘N to open this dialog
        </p>
      </form>
    </Modal>
  )
}

function PersonaCard({
  persona,
  selected,
  onClick
}: {
  persona: Persona
  selected: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`p-3 text-left rounded transition-all ${
        selected
          ? 'bg-gray-800 ring-1 ring-blue-500'
          : 'bg-gray-800/50 hover:bg-gray-800'
      }`}
    >
      <div className="flex items-center gap-2">
        <span
          className="w-2 h-2 rounded-full"
          style={{ backgroundColor: persona.color }}
        />
        <span className="text-xs font-medium text-gray-200">
          {persona.name}
        </span>
      </div>
      <p className="mt-1 text-[10px] text-gray-500 line-clamp-2">
        {persona.description}
      </p>
    </button>
  )
}
