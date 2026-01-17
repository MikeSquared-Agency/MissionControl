import { useState } from 'react'
import { Modal } from './Modal'
import { useStore } from '../stores/useStore'
import type { Agent } from '../types'

interface MoveAgentDialogProps {
  open: boolean
  onClose: () => void
  agent: Agent | null
}

export function MoveAgentDialog({ open, onClose, agent }: MoveAgentDialogProps) {
  const zones = useStore((s) => s.zones)
  const updateAgent = useStore((s) => s.updateAgent)

  const [selectedZone, setSelectedZone] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Filter out current zone
  const availableZones = zones.filter((z) => z.id !== agent?.zone)

  const handleMove = async () => {
    if (!agent || !selectedZone) return

    setLoading(true)
    setError('')

    try {
      // Update via API
      const res = await fetch(`/api/agents/${agent.id}/move`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ zoneId: selectedZone })
      })

      if (!res.ok) {
        throw new Error(await res.text())
      }

      // Update local state
      updateAgent(agent.id, { zone: selectedZone })
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to move agent')
    } finally {
      setLoading(false)
    }
  }

  if (!agent) return null

  const currentZone = zones.find((z) => z.id === agent.zone)

  return (
    <Modal open={open} onClose={onClose} title="Move Agent" width="sm">
      <div className="space-y-4">
        {/* Agent info */}
        <div className="px-3 py-2 bg-gray-800/50 rounded">
          <p className="text-sm text-gray-300">{agent.name}</p>
          <p className="text-xs text-gray-500 mt-0.5">
            Currently in: {currentZone?.name || 'Unknown'}
          </p>
        </div>

        {/* Zone selection */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Move to zone
          </label>
          {availableZones.length === 0 ? (
            <p className="text-xs text-gray-500 italic">
              No other zones available. Create a new zone first.
            </p>
          ) : (
            <div className="space-y-1">
              {availableZones.map((zone) => (
                <button
                  key={zone.id}
                  type="button"
                  onClick={() => setSelectedZone(zone.id)}
                  className={`w-full flex items-center gap-2 px-3 py-2 rounded transition-colors text-left ${
                    selectedZone === zone.id
                      ? 'bg-blue-600/20 border border-blue-500/50'
                      : 'bg-gray-800 hover:bg-gray-700 border border-transparent'
                  }`}
                >
                  <span
                    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: zone.color }}
                  />
                  <span className="text-sm text-gray-200">{zone.name}</span>
                  {zone.workingDir && (
                    <span className="text-[10px] font-mono text-gray-600 truncate">
                      {zone.workingDir}
                    </span>
                  )}
                </button>
              ))}
            </div>
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
            onClick={handleMove}
            disabled={loading || !selectedZone || availableZones.length === 0}
            className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors"
          >
            {loading ? 'Moving...' : 'Move Agent'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
