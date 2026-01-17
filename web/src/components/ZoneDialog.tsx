import { useState, useEffect } from 'react'
import { Modal } from './Modal'
import { Spinner } from './Spinner'
import { useStore, createZone, updateZoneApi } from '../stores/useStore'
import { toast } from '../stores/useToast'
import type { Zone } from '../types'

interface ZoneDialogProps {
  open: boolean
  onClose: () => void
  mode: 'create' | 'edit' | 'duplicate'
  zone?: Zone
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

export function ZoneDialog({ open, onClose, mode, zone }: ZoneDialogProps) {
  const addZone = useStore((s) => s.addZone)
  const updateZone = useStore((s) => s.updateZone)

  const [name, setName] = useState('')
  const [color, setColor] = useState('#6b7280')
  const [workingDir, setWorkingDir] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      if (mode === 'create') {
        setName('')
        setColor('#6b7280')
        setWorkingDir('')
      } else if (zone) {
        setName(mode === 'duplicate' ? `${zone.name}-copy` : zone.name)
        setColor(zone.color)
        setWorkingDir(zone.workingDir)
      }
      setError('')
    }
  }, [open, mode, zone])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return

    setLoading(true)
    setError('')

    try {
      const zoneName = name.trim()
      if (mode === 'edit' && zone) {
        // Update existing zone
        const updated = await updateZoneApi(zone.id, {
          name: zoneName,
          color,
          workingDir: workingDir.trim()
        })
        updateZone(zone.id, updated)
        toast.success(`Zone "${zoneName}" updated`)
      } else {
        // Create new zone (create or duplicate)
        const created = await createZone({
          name: zoneName,
          color,
          workingDir: workingDir.trim()
        })
        addZone(created)
        toast.success(`Zone "${zoneName}" created`)
      }
      onClose()
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to save zone'
      setError(errorMsg)
      toast.error(errorMsg)
    } finally {
      setLoading(false)
    }
  }

  const title = mode === 'create' ? 'New Zone' : mode === 'edit' ? 'Edit Zone' : 'Duplicate Zone'

  return (
    <Modal open={open} onClose={onClose} title={title} width="sm">
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Name */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Zone Name <span className="text-red-400">*</span>
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Frontend, Backend, Tests"
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
            autoFocus
          />
        </div>

        {/* Color */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">Color</label>
          <div className="flex items-center gap-2">
            <div className="flex gap-1">
              {PRESET_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setColor(c)}
                  className={`w-6 h-6 rounded transition-all ${
                    color === c ? 'ring-2 ring-white ring-offset-2 ring-offset-gray-900' : ''
                  }`}
                  style={{ backgroundColor: c }}
                />
              ))}
            </div>
            <input
              type="text"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              placeholder="#hex"
              className="w-20 px-2 py-1 text-xs font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
            />
          </div>
        </div>

        {/* Working Directory */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Default Working Directory
          </label>
          <input
            type="text"
            value={workingDir}
            onChange={(e) => setWorkingDir(e.target.value)}
            placeholder="/path/to/project"
            className="w-full px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
          />
          <p className="mt-1 text-[10px] text-gray-600">
            Agents spawned in this zone will use this directory by default
          </p>
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
            disabled={loading || !name.trim()}
            className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors flex items-center justify-center gap-2"
          >
            {loading && <Spinner size="sm" />}
            {loading ? 'Saving...' : mode === 'edit' ? 'Save Changes' : 'Create Zone'}
          </button>
        </div>
      </form>
    </Modal>
  )
}
