import { useState, useEffect } from 'react'
import { Modal } from './Modal'
import { Spinner } from './Spinner'
import { useStore } from '../stores/useStore'
import { toast } from '../stores/useToast'
import type { Zone } from '../types'

interface MergeZoneDialogProps {
  open: boolean
  onClose: () => void
  zone: Zone | null
}

export function MergeZoneDialog({ open, onClose, zone }: MergeZoneDialogProps) {
  const zones = useStore((s) => s.zones)
  const agents = useStore((s) => s.agents)
  const removeZone = useStore((s) => s.removeZone)
  const updateAgent = useStore((s) => s.updateAgent)

  const [targetZoneId, setTargetZoneId] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Other zones to merge into
  const targetZones = zones.filter((z) => z.id !== zone?.id)

  // Agents in source zone
  const zoneAgents = agents.filter((a) => a.zone === zone?.id)

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setTargetZoneId(null)
      setError('')
    }
  }, [open])

  const handleMerge = async () => {
    if (!zone || !targetZoneId) return

    setLoading(true)
    setError('')

    try {
      // Move all agents to target zone
      for (const agent of zoneAgents) {
        const res = await fetch(`/api/agents/${agent.id}/move`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ zoneId: targetZoneId })
        })

        if (!res.ok) {
          throw new Error(await res.text())
        }

        updateAgent(agent.id, { zone: targetZoneId })
      }

      // Delete the source zone
      const res = await fetch(`/api/zones/${zone.id}`, {
        method: 'DELETE'
      })

      if (!res.ok) {
        throw new Error(await res.text())
      }

      removeZone(zone.id)
      toast.success(`Zone "${zone.name}" merged and deleted`)
      onClose()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to merge zone'
      setError(errorMsg)
      toast.error(errorMsg)
    } finally {
      setLoading(false)
    }
  }

  if (!zone) return null

  const targetZone = targetZones.find((z) => z.id === targetZoneId)

  return (
    <Modal open={open} onClose={onClose} title="Merge Zone" width="sm">
      <div className="space-y-4">
        {/* Source zone info */}
        <div className="px-3 py-2 bg-gray-800/50 rounded">
          <p className="text-xs text-gray-500">Merge and delete</p>
          <div className="flex items-center gap-2 mt-0.5">
            <span
              className="w-2.5 h-2.5 rounded-full flex-shrink-0"
              style={{ backgroundColor: zone.color }}
            />
            <p className="text-sm text-gray-200">{zone.name}</p>
            <span className="text-xs text-gray-600">({zoneAgents.length} agents)</span>
          </div>
        </div>

        {/* Target zone selection */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Move agents into
          </label>
          {targetZones.length === 0 ? (
            <p className="text-xs text-gray-600 italic py-2">
              No other zones available. Create another zone first.
            </p>
          ) : (
            <div className="space-y-1">
              {targetZones.map((z) => (
                <button
                  key={z.id}
                  type="button"
                  onClick={() => setTargetZoneId(z.id)}
                  className={`w-full flex items-center gap-2 px-3 py-2 rounded transition-colors text-left ${
                    targetZoneId === z.id
                      ? 'bg-blue-600/20 border border-blue-500/50'
                      : 'bg-gray-800 hover:bg-gray-700 border border-transparent'
                  }`}
                >
                  <span
                    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: z.color }}
                  />
                  <span className="text-sm text-gray-200">{z.name}</span>
                  {z.workingDir && (
                    <span className="text-[10px] font-mono text-gray-600 truncate">
                      {z.workingDir}
                    </span>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Preview */}
        {targetZone && zoneAgents.length > 0 && (
          <div className="px-3 py-2 bg-amber-500/10 border border-amber-500/20 rounded">
            <p className="text-xs text-amber-400">
              {zoneAgents.length} agent{zoneAgents.length !== 1 ? 's' : ''} will be moved to "{targetZone.name}".
              Zone "{zone.name}" will be deleted.
            </p>
          </div>
        )}

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
            onClick={handleMerge}
            disabled={loading || !targetZoneId || targetZones.length === 0}
            className="flex-1 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-500 disabled:bg-red-800 disabled:cursor-not-allowed rounded transition-colors flex items-center justify-center gap-2"
          >
            {loading && <Spinner size="sm" />}
            {loading ? 'Merging...' : 'Merge & Delete'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
