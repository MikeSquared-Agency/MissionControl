import { useState } from 'react'
import {
  useKnowledgeStore,
  useBudgets,
  useBudgetAlerts,
  useTotalTokenUsage,
  useCheckpoints,
  useRecentHandoffs,
  createBudget,
  createCheckpoint
} from '../../stores/useKnowledgeStore'
import type { TokenBudget, BudgetStatus, CheckpointSummary, Handoff } from '../../types/v4'
import { getBudgetStatusColor } from '../../types/v4'

export function TokenUsage() {
  const budgets = useBudgets()
  const alerts = useBudgetAlerts()
  const totals = useTotalTokenUsage()
  const checkpoints = useCheckpoints()
  const handoffs = useRecentHandoffs()
  const [showNewBudget, setShowNewBudget] = useState(false)

  return (
    <div className="h-full flex flex-col bg-gray-900">
      {/* Header with totals */}
      <div className="p-3 bg-gray-850 border-b border-gray-800">
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-sm font-semibold text-gray-200">Token Usage</h2>
          <button
            onClick={() => setShowNewBudget(true)}
            className="px-2 py-1 text-[10px] font-medium text-gray-400 hover:text-gray-300 hover:bg-gray-800 rounded transition-colors"
          >
            + Budget
          </button>
        </div>

        {/* Total usage bar */}
        <TotalUsageBar used={totals.used} budget={totals.budget} workers={totals.workers} />

        {/* Alerts summary */}
        {alerts.length > 0 && (
          <div className="mt-2 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
            <span className="text-xs text-amber-400">
              {alerts.length} worker{alerts.length !== 1 ? 's' : ''} need attention
            </span>
          </div>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-3 space-y-4">
        {/* New budget form */}
        {showNewBudget && (
          <NewBudgetForm onClose={() => setShowNewBudget(false)} />
        )}

        {/* Worker budgets */}
        <section>
          <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
            Worker Budgets
          </h3>
          {Object.keys(budgets).length === 0 ? (
            <div className="text-sm text-gray-600 py-4 text-center">
              No active budgets
            </div>
          ) : (
            <div className="space-y-2">
              {Object.values(budgets).map((budget) => (
                <BudgetCard key={budget.worker_id} budget={budget} />
              ))}
            </div>
          )}
        </section>

        {/* Checkpoints */}
        <section>
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Checkpoints
            </h3>
            <CreateCheckpointButton />
          </div>
          {checkpoints.length === 0 ? (
            <div className="text-sm text-gray-600 py-4 text-center">
              No checkpoints yet
            </div>
          ) : (
            <div className="space-y-1">
              {checkpoints.slice(-5).reverse().map((cp) => (
                <CheckpointRow key={cp.id} checkpoint={cp} />
              ))}
            </div>
          )}
        </section>

        {/* Recent handoffs */}
        <section>
          <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
            Recent Handoffs
          </h3>
          {handoffs.length === 0 ? (
            <div className="text-sm text-gray-600 py-4 text-center">
              No handoffs yet
            </div>
          ) : (
            <div className="space-y-1">
              {handoffs.slice(0, 5).map((h, i) => (
                <HandoffRow key={`${h.task_id}-${i}`} handoff={h} />
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

interface TotalUsageBarProps {
  used: number
  budget: number
  workers: number
}

function TotalUsageBar({ used, budget, workers }: TotalUsageBarProps) {
  const percentage = budget > 0 ? Math.min((used / budget) * 100, 100) : 0
  const status: BudgetStatus =
    percentage >= 100 ? 'exceeded' :
    percentage >= 75 ? 'critical' :
    percentage >= 50 ? 'warning' : 'healthy'

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-xs">
        <span className="text-gray-500">{workers} worker{workers !== 1 ? 's' : ''}</span>
        <span className="font-mono text-gray-400">
          {formatTokens(used)} / {formatTokens(budget)}
        </span>
      </div>
      <div className="h-1.5 bg-gray-800 rounded-full overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-300"
          style={{
            width: `${percentage}%`,
            backgroundColor: getBudgetStatusColor(status)
          }}
        />
      </div>
    </div>
  )
}

interface BudgetCardProps {
  budget: TokenBudget
}

function BudgetCard({ budget }: BudgetCardProps) {
  const percentage = budget.budget > 0
    ? Math.min((budget.used / budget.budget) * 100, 100)
    : 0

  return (
    <div className={`
      p-2.5 rounded-lg border transition-colors
      ${budget.status === 'exceeded'
        ? 'bg-red-500/10 border-red-500/30'
        : budget.status === 'critical'
          ? 'bg-red-500/5 border-red-500/20'
          : budget.status === 'warning'
            ? 'bg-amber-500/5 border-amber-500/20'
            : 'bg-gray-800/30 border-gray-800'
      }
    `}>
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span
            className={`w-2 h-2 rounded-full ${
              budget.status === 'healthy' ? '' : 'animate-pulse'
            }`}
            style={{ backgroundColor: getBudgetStatusColor(budget.status) }}
          />
          <span className="text-sm font-medium text-gray-200 truncate">
            {budget.worker_id}
          </span>
        </div>
        <span
          className="px-1.5 py-0.5 text-[10px] font-medium rounded"
          style={{
            backgroundColor: `${getBudgetStatusColor(budget.status)}20`,
            color: getBudgetStatusColor(budget.status)
          }}
        >
          {budget.status}
        </span>
      </div>

      {/* Progress bar */}
      <div className="h-1 bg-gray-700 rounded-full overflow-hidden mb-1.5">
        <div
          className="h-full rounded-full transition-all duration-300"
          style={{
            width: `${percentage}%`,
            backgroundColor: getBudgetStatusColor(budget.status)
          }}
        />
      </div>

      {/* Stats */}
      <div className="flex items-center justify-between text-[10px]">
        <span className="text-gray-500">
          {percentage.toFixed(0)}% used
        </span>
        <span className="font-mono text-gray-400">
          {formatTokens(budget.used)} / {formatTokens(budget.budget)}
        </span>
        <span className="text-gray-500">
          {formatTokens(budget.remaining)} left
        </span>
      </div>
    </div>
  )
}

interface NewBudgetFormProps {
  onClose: () => void
}

function NewBudgetForm({ onClose }: NewBudgetFormProps) {
  const [workerID, setWorkerID] = useState('')
  const [budget, setBudget] = useState(20000)
  const [submitting, setSubmitting] = useState(false)
  const setBudgetStore = useKnowledgeStore((s) => s.setBudget)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!workerID.trim()) return

    setSubmitting(true)
    try {
      const result = await createBudget(workerID.trim(), budget)
      setBudgetStore(workerID.trim(), result)
      onClose()
    } catch (err) {
      console.error('Failed to create budget:', err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="p-3 rounded-lg bg-gray-800/50 border border-gray-700">
      <div className="flex items-center justify-between mb-3">
        <h4 className="text-xs font-medium text-gray-300">New Budget</h4>
        <button
          type="button"
          onClick={onClose}
          className="text-gray-500 hover:text-gray-400"
        >
          ✕
        </button>
      </div>

      <div className="space-y-2">
        <input
          type="text"
          value={workerID}
          onChange={(e) => setWorkerID(e.target.value)}
          placeholder="Worker ID"
          className="w-full px-2 py-1.5 text-xs bg-gray-900 border border-gray-700 rounded text-gray-300 placeholder-gray-600 focus:outline-none focus:border-gray-600"
        />

        <div className="flex items-center gap-2">
          <input
            type="number"
            value={budget}
            onChange={(e) => setBudget(Number(e.target.value))}
            min={1000}
            step={1000}
            className="flex-1 px-2 py-1.5 text-xs bg-gray-900 border border-gray-700 rounded text-gray-300 focus:outline-none focus:border-gray-600"
          />
          <span className="text-xs text-gray-500">tokens</span>
        </div>

        <button
          type="submit"
          disabled={submitting || !workerID.trim()}
          className="w-full py-1.5 text-xs font-medium bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded transition-colors"
        >
          {submitting ? 'Creating...' : 'Create Budget'}
        </button>
      </div>
    </form>
  )
}

function CreateCheckpointButton() {
  const [creating, setCreating] = useState(false)
  const addCheckpoint = useKnowledgeStore((s) => s.addCheckpoint)

  const handleCreate = async () => {
    setCreating(true)
    try {
      const checkpoint = await createCheckpoint()
      addCheckpoint(checkpoint)
    } catch (err) {
      console.error('Failed to create checkpoint:', err)
    } finally {
      setCreating(false)
    }
  }

  return (
    <button
      onClick={handleCreate}
      disabled={creating}
      className="px-2 py-0.5 text-[10px] font-medium text-gray-500 hover:text-gray-400 hover:bg-gray-800 rounded transition-colors disabled:opacity-50"
    >
      {creating ? 'Creating...' : '+ Checkpoint'}
    </button>
  )
}

interface CheckpointRowProps {
  checkpoint: CheckpointSummary
}

function CheckpointRow({ checkpoint }: CheckpointRowProps) {
  return (
    <div className="flex items-center justify-between px-2 py-1.5 rounded bg-gray-800/30 text-xs">
      <div className="flex items-center gap-2">
        <span className="text-gray-500">📍</span>
        <span className="font-mono text-gray-400">{checkpoint.id}</span>
      </div>
      <div className="flex items-center gap-2">
        <span className="px-1.5 py-0.5 text-[10px] font-medium text-gray-500 bg-gray-800 rounded">
          {checkpoint.phase}
        </span>
        <span className="text-[10px] text-gray-600">
          {formatTime(checkpoint.created_at)}
        </span>
      </div>
    </div>
  )
}

interface HandoffRowProps {
  handoff: Handoff
}

function HandoffRow({ handoff }: HandoffRowProps) {
  const statusColors: Record<string, string> = {
    complete: 'text-green-500',
    blocked: 'text-red-500',
    partial: 'text-amber-500'
  }

  return (
    <div className="flex items-center justify-between px-2 py-1.5 rounded bg-gray-800/30 text-xs">
      <div className="flex items-center gap-2">
        <span className={statusColors[handoff.status] || 'text-gray-500'}>
          {handoff.status === 'complete' ? '✓' : handoff.status === 'blocked' ? '✕' : '◐'}
        </span>
        <span className="text-gray-400 truncate max-w-[100px]">{handoff.task_id}</span>
      </div>
      <div className="flex items-center gap-2">
        <span className="text-gray-500 truncate max-w-[80px]">{handoff.worker_id}</span>
        <span className="text-[10px] text-gray-600">
          {formatTime(handoff.timestamp)}
        </span>
      </div>
    </div>
  )
}

function formatTokens(tokens: number): string {
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(1)}M`
  }
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}k`
  }
  return String(tokens)
}

function formatTime(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`

  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`

  return date.toLocaleDateString()
}
