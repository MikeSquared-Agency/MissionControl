import { useState } from 'react'
import { Modal } from './Modal'
import { useStore, sendMessage } from '../stores/useStore'
import type { Agent } from '../types'

interface ContextShareDialogProps {
  open: boolean
  onClose: () => void
  sourceAgent: Agent | null
}

export function ContextShareDialog({ open, onClose, sourceAgent }: ContextShareDialogProps) {
  const agents = useStore((s) => s.agents)

  const [selectedFindings, setSelectedFindings] = useState<Set<number>>(new Set())
  const [selectedTargets, setSelectedTargets] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  // Get other active agents
  const targetAgents = agents.filter(
    (a) => a.id !== sourceAgent?.id && a.status !== 'stopped'
  )

  const toggleFinding = (index: number) => {
    const next = new Set(selectedFindings)
    if (next.has(index)) {
      next.delete(index)
    } else {
      next.add(index)
    }
    setSelectedFindings(next)
  }

  const toggleTarget = (agentId: string) => {
    const next = new Set(selectedTargets)
    if (next.has(agentId)) {
      next.delete(agentId)
    } else {
      next.add(agentId)
    }
    setSelectedTargets(next)
  }

  const handleShare = async () => {
    if (!sourceAgent || selectedFindings.size === 0 || selectedTargets.size === 0) return

    setLoading(true)
    setError('')

    try {
      // Build the context message
      const findings = Array.from(selectedFindings)
        .map((i) => sourceAgent.findings[i])
        .filter(Boolean)

      const message = `Context from ${sourceAgent.name}:\n\n${findings.map((f) => `• ${f}`).join('\n')}`

      // Send to each target agent
      for (const targetId of selectedTargets) {
        await sendMessage(targetId, message)
      }

      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to share context')
    } finally {
      setLoading(false)
    }
  }

  if (!sourceAgent) return null

  const hasFindings = sourceAgent.findings.length > 0
  const hasTargets = targetAgents.length > 0
  const canShare = selectedFindings.size > 0 && selectedTargets.size > 0

  return (
    <Modal open={open} onClose={onClose} title="Share Context" width="md">
      <div className="space-y-4">
        {/* Source agent info */}
        <div className="px-3 py-2 bg-gray-800/50 rounded">
          <p className="text-xs text-gray-500">Sharing from</p>
          <p className="text-sm text-gray-200">{sourceAgent.name}</p>
        </div>

        {/* Findings to share */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Findings to share
          </label>
          {!hasFindings ? (
            <p className="text-xs text-gray-600 italic py-2">
              No findings recorded yet. The agent hasn't reported any discoveries.
            </p>
          ) : (
            <div className="space-y-1 max-h-32 overflow-y-auto">
              {sourceAgent.findings.map((finding, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => toggleFinding(i)}
                  className={`w-full flex items-start gap-2 px-3 py-2 rounded text-left transition-colors ${
                    selectedFindings.has(i)
                      ? 'bg-blue-600/20 border border-blue-500/50'
                      : 'bg-gray-800 hover:bg-gray-700 border border-transparent'
                  }`}
                >
                  <span className={`mt-0.5 ${selectedFindings.has(i) ? 'text-blue-400' : 'text-gray-600'}`}>
                    {selectedFindings.has(i) ? '✓' : '○'}
                  </span>
                  <span className="text-xs text-gray-300">{finding}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Target agents */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1.5">
            Share with
          </label>
          {!hasTargets ? (
            <p className="text-xs text-gray-600 italic py-2">
              No other active agents available.
            </p>
          ) : (
            <div className="space-y-1 max-h-32 overflow-y-auto">
              {targetAgents.map((agent) => (
                <button
                  key={agent.id}
                  type="button"
                  onClick={() => toggleTarget(agent.id)}
                  className={`w-full flex items-center gap-2 px-3 py-2 rounded text-left transition-colors ${
                    selectedTargets.has(agent.id)
                      ? 'bg-blue-600/20 border border-blue-500/50'
                      : 'bg-gray-800 hover:bg-gray-700 border border-transparent'
                  }`}
                >
                  <span
                    className={`w-2 h-2 rounded-full ${
                      agent.status === 'working' ? 'bg-green-500' :
                      agent.status === 'waiting' ? 'bg-amber-500' :
                      'bg-gray-500'
                    }`}
                  />
                  <span className="text-sm text-gray-200">{agent.name}</span>
                  <span className="text-[10px] text-gray-600">{agent.status}</span>
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
            onClick={handleShare}
            disabled={loading || !canShare}
            className="flex-1 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:cursor-not-allowed rounded transition-colors"
          >
            {loading ? 'Sharing...' : `Share with ${selectedTargets.size} agent${selectedTargets.size !== 1 ? 's' : ''}`}
          </button>
        </div>
      </div>
    </Modal>
  )
}
