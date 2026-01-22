import { useState, useEffect } from 'react'
import { useStore, spawnAgent } from '../stores/useStore'
import { useProjectStore } from '../stores/useProjectStore'
import { toast } from '../stores/useToast'
import { Modal } from './Modal'
import { Spinner } from './Spinner'
import type { Persona } from '../types'

interface SpawnDialogProps {
  open: boolean
  onClose: () => void
}

export function SpawnDialog({ open, onClose }: SpawnDialogProps) {
  const personas = useStore((s) => s.personas)
  const zones = useStore((s) => s.zones)
  const addAgent = useStore((s) => s.addAgent)
  const selectAgent = useStore((s) => s.selectAgent)

  // Get current project for offline mode
  const currentProjectPath = useProjectStore((s) => s.currentProject)
  const projects = useProjectStore((s) => s.projects)
  const currentProject = projects.find((p) => p.path === currentProjectPath)

  const [selectedPersona, setSelectedPersona] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [task, setTask] = useState('')
  const [zone, setZone] = useState('default')
  const [workingDir, setWorkingDir] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setSelectedPersona(null)
      setName('')
      setTask('')
      setZone('default')
      setWorkingDir(currentProject?.path || '')
      setError('')
    }
  }, [open, currentProject?.path])

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
        name: agentName,
        task: task.trim(),
        persona: selectedPersona || undefined,
        zone,
        workingDir: workingDir.trim() || currentProject?.path || undefined,
        offlineMode: currentProject?.mode === 'offline',
        ollamaModel: currentProject?.ollamaModel
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
        {/* Mode indicator */}
        {currentProject && (
          <div className="flex items-center gap-2 text-xs text-gray-500 pb-2 border-b border-gray-800">
            <span className={`w-2 h-2 rounded-full ${
              currentProject.mode === 'offline' ? 'bg-yellow-500' : 'bg-green-500'
            }`} />
            {currentProject.mode === 'offline'
              ? `Offline (${currentProject.ollamaModel || 'Ollama'})`
              : 'Online (Claude API)'}
          </div>
        )}

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
