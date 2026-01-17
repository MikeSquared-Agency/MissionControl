import { useState, useEffect } from 'react'
import { Modal } from './Modal'
import { useStore, createZone } from '../stores/useStore'
import type { Zone } from '../types'

interface SplitZoneDialogProps {
  open: boolean
  onClose: () => void
  zone: Zone | null
}

const PRESET_COLORS = [
  '#6b7280', // gray
  '#ef4444', // red
  '#f97316', // orange
  '#eab308', // yellow
  '#22c55e', // green
  '#14b8a6', // teal
  '#3b82f6', // blue
  '#8b5cf6', // violet
  '#ec4899', // pink
]

export function SplitZoneDialog({ open, onClose, zone }: SplitZoneDialogProps) {
  const agents = useStore((s) => s.agents)
  const addZone = useStore((s) => s.addZone)
  const updateAgent = useStore((s) => s.updateAgent)

  const [newZoneName, setNewZoneName] = useState('')
  const [newZoneColor, setNewZoneColor] = useState('#3b82f6')
  const [selectedAgents, setSelectedAgents] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Agents in this zone
  const zoneAgents = agents.filter((a) => a.zone === zone?.id)

  // Reset form when dialog opens
  useEffect(() => {
    if (open && zone) {
      setNewZoneName('')
      setNewZoneColor('#3b82f6')
      setSelectedAgents(new Set())
      setError('')
    }
  }, [open, zone])

  const toggleAgent = (agentId: string) => {
    const next = new Set(selectedAgents)
    if (next.has(agentId)) {
      next.delete(agentId)
    } else {
      next.add(agentId)
    }
    setSelectedAgents(next)
  }

  const handleSplit = async () => {
    if (!zone || !newZoneName.trim() || selectedAgents.size === 0) return

    setLoading(true)
    setError('')

    try {
      // Create the new zone
      const created = await createZone({
        name: newZoneName.trim(),
        color: newZoneColor,
        workingDir: zone.workingDir
      })
      addZone(created)

      // Move selected agents to new zone
      for (const agentId of selectedAgents) {
        const res = await fetch(`/api/agents/${agentId}/move`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ zoneId: created.id })
        })

        if (!res.ok) {
          throw new Error(await res.text())
        }

        updateAgent(agentId, { zone: created.id })
      }

      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to split zone')
    } finally {
      setLoading(false)
    }
  }

  if (!zone) return null

  const canSplit = newZoneName.trim() && selectedAgents.size > 0 && selectedAgents.size < zoneAgents.length

  return (
    <Modal open={open} onClose={onClose} title="Split Zone" width="md">
      <div className="space-y-4">
        {/* Source zone info */}
        <div className="px-3 py-2 bg-gray-800/50 rounded">
          <p className="text-xs text-gray-500">Splitting from</p>
          <div className="flex items-center gap-2 mt-0.5">
            <span
              className="w-2.5 h-2.5 rounded-full flex-shrink-0"
              style={{ backgroundColor: zone.color }}
            />
            <p className="text-sm text-gray-200">{zone.name}</p>
            <span className="text-xs text-gray-600">({zoneAgents.length} agents)</span>
          </div>
        </div>

        {/* New zone name */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            New Zone Name <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            value={newZoneName}
            onChange={(e) => setNewZoneName(e.target.value)}
            placeholder="e.g., Backend Tests"
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* New zone color */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">Color</label>
          <div className="flex items-center gap-2">
            <div className="flex gap-1">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setNewZoneColor(c)}
                  className={`w-6 h-6 rounded transition-all ${
                    newZoneColor === c ? 'ring-2 ring-white ring-offset-2 ring-offset-gray-900' : ''
                  }`}
                  style={{ backgroundColor: c }}
                />
              ))}
            </div>
          </div>
        </div>

        {/* Agent selection */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Agents to move to new zone
          </label>
          {zoneAgents.length === 0 ? (
            <p className="text-xs text-gray-600 italic py-2">
              No agents in this zone.
            </p>
          ) : zoneAgents.length === 1 ? (
            <p className="text-xs text-gray-600 italic py-2">
              Only one agent in this zone. Need at least two agents to split.
            </p>
          ) : (
            <div className="space-y-1 max-h-40 overflow-y-auto">
              {zoneAgents.map((agent) => (
                <button
                  key={agent.id}
                  type="button"
                  onClick={() => toggleAgent(agent.id)}
                  className={`w-full flex items-center gap-2 px-3 py-2 rounded transition-colors text-left ${
                    selectedAgents.has(agent.id)
                      ? 'bg-blue-600/20 border border-blue-500/50'
                      : 'bg-gray-800 hover:bg-gray-700 border border-transparent'
                  }`}
                >
                  <span className={`${selectedAgents.has(agent.id) ? 'text-blue-400' : 'text-gray-600'}`}>
                    {selectedAgents.has(agent.id) ? '✓' : '○'}
                  </span>
                  <span
                    className={`w-2 h-2 rounded-full ${
                      agent.status === 'working' ? 'bg-green-500' :
                      agent.status === 'waiting' ? 'bg-amber-500' :
                      agent.status === 'error' ? 'bg-red-500' :
                      'bg-gray-500'
                    }`}
                  />
                  <span className="text-sm text-gray-200">{agent.name}</span>
                  <span className="text-[10px] text-gray-600">{agent.status}</span>
                </button>
              ))}
            </div>
          )}
          {selectedAgents.size === zoneAgents.length && zoneAgents.length > 0 && (
            <p className="text-[10px] text-amber-500 mt-1">
              Leave at least one agent in the original zone
            </p>
          )}
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
            onClick={handleSplit}
            disabled={loading || !canSplit}
            className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors"
          >
            {loading ? 'Splitting...' : `Split ${selectedAgents.size} agent${selectedAgents.size !== 1 ? 's' : ''}`}
          </button>
        </div>
      </div>
    </Modal>
  )
}
