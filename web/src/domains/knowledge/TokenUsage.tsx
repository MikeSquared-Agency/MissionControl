import { useState, useEffect } from 'react'
import {
  useKnowledgeStore,
  useBudgets,
  useBudgetAlerts,
  useTotalTokenUsage,
  useCheckpoints,
  useRecentHandoffs,
  useSessionStatus,
  useSessionHistory,
  createBudget,
  createCheckpoint,
  fetchSessionStatus,
  fetchSessionHistory,
  restartSession
} from '../../stores/useKnowledgeStore'
import type { TokenBudget, BudgetStatus, CheckpointSummary, Handoff, SessionStatus, SessionRecord } from '../../types/workflow'
import { getBudgetStatusColor, getStageLabel } from '../../types/workflow'
import { toast } from '../../stores/useToast'

export function TokenUsage() {
  const budgets = useBudgets()
  const alerts = useBudgetAlerts()
  const totals = useTotalTokenUsage()
  const checkpoints = useCheckpoints()
  const handoffs = useRecentHandoffs()
  const sessionStatus = useSessionStatus()
  const sessionHistory = useSessionHistory()
  const setSessionStatus = useKnowledgeStore((s) => s.setSessionStatus)
  const setSessionHistory = useKnowledgeStore((s) => s.setSessionHistory)
  const [showNewBudget, setShowNewBudget] = useState(false)
  const [showHistory, setShowHistory] = useState(false)

  // Fetch session status on mount and periodically
  useEffect(() => {
    const load = () => {
      fetchSessionStatus()
        .then(setSessionStatus)
        .catch(() => {}) // Silently fail if endpoint unavailable
    }
    load()
    const interval = setInterval(load, 30000)
    return () => clearInterval(interval)
  }, [setSessionStatus])

  // Fetch session history when expanded
  useEffect(() => {
    if (showHistory) {
      fetchSessionHistory()
        .then(setSessionHistory)
        .catch(() => {})
    }
  }, [showHistory, setSessionHistory])

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
        {/* Session health indicator (G5.1) */}
        {sessionStatus && (
          <SessionHealthCard status={sessionStatus} />
        )}

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

        {/* Checkpoints (G5.3) */}
        <section>
          <div className="flex items-center justify-between mb-2">
            <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider">
              Checkpoints
            </h3>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setShowHistory(!showHistory)}
                className="px-2 py-0.5 text-[10px] font-medium text-gray-500 hover:text-gray-400 hover:bg-gray-800 rounded transition-colors"
              >
                {showHistory ? 'Hide History' : 'Session History'}
              </button>
              <CreateCheckpointButton />
            </div>
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

        {/* Session history (G5.3) */}
        {showHistory && (
          <section>
            <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
              Session History
            </h3>
            {sessionHistory.length === 0 ? (
              <div className="text-sm text-gray-600 py-4 text-center">
                No previous sessions
              </div>
            ) : (
              <div className="space-y-1">
                {sessionHistory.slice().reverse().map((s) => (
                  <SessionRow key={s.session_id} session={s} />
                ))}
              </div>
            )}
          </section>
        )}

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

// ============================================================================
// Session Health Card (G5.1)
// ============================================================================

interface SessionHealthCardProps {
  status: SessionStatus
}

function SessionHealthCard({ status }: SessionHealthCardProps) {
  const [confirmRestart, setConfirmRestart] = useState(false)
  const [restarting, setRestarting] = useState(false)
  const setSessionStatus = useKnowledgeStore((s) => s.setSessionStatus)

  const healthColors: Record<string, { bg: string; border: string; dot: string; text: string }> = {
    green: { bg: 'bg-green-500/5', border: 'border-green-500/20', dot: 'bg-green-500', text: 'text-green-400' },
    yellow: { bg: 'bg-amber-500/5', border: 'border-amber-500/20', dot: 'bg-amber-500', text: 'text-amber-400' },
    red: { bg: 'bg-red-500/5', border: 'border-red-500/20', dot: 'bg-red-500', text: 'text-red-400' },
  }

  const colors = healthColors[status.health] || healthColors.green

  const handleRestart = async () => {
    setRestarting(true)
    try {
      const result = await restartSession()
      toast.success(`Session restarted. New session: ${result.new_session_id.slice(0, 8)}...`)
      // Refresh session status
      fetchSessionStatus()
        .then(setSessionStatus)
        .catch(() => {})
    } catch (err) {
      toast.error('Failed to restart session')
      console.error('Failed to restart session:', err)
    } finally {
      setRestarting(false)
      setConfirmRestart(false)
    }
  }

  return (
    <section>
      <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
        Session Health
      </h3>
      <div className={`p-3 rounded-lg border ${colors.bg} ${colors.border}`}>
        {/* Health row */}
        <div className="flex items-center justify-between mb-2">
          <div className="flex items-center gap-2">
            <span className={`w-2.5 h-2.5 rounded-full ${colors.dot} ${status.health !== 'green' ? 'animate-pulse' : ''}`} />
            <span className="text-sm font-medium text-gray-200">
              {getStageLabel(status.stage)}
            </span>
          </div>
          <span className={`text-xs font-medium ${colors.text}`}>
            {status.health === 'green' ? 'Healthy' : status.health === 'yellow' ? 'Caution' : 'Critical'}
          </span>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-2 mb-2 text-[10px]">
          <div>
            <span className="text-gray-500 block">Duration</span>
            <span className="text-gray-300 font-mono">{formatDuration(status.duration_minutes)}</span>
          </div>
          <div>
            <span className="text-gray-500 block">Tasks</span>
            <span className="text-gray-300 font-mono">{status.tasks_complete}/{status.tasks_total}</span>
          </div>
          <div>
            <span className="text-gray-500 block">Session</span>
            <span className="text-gray-300 font-mono">{status.session_id.slice(0, 8)}</span>
          </div>
        </div>

        {/* Recommendation */}
        {status.recommendation && (
          <p className="text-[10px] text-gray-500 mb-2">{status.recommendation}</p>
        )}

        {/* Restart button (G5.2) */}
        {!confirmRestart ? (
          <button
            onClick={() => setConfirmRestart(true)}
            className="w-full py-1.5 text-xs font-medium text-gray-400 hover:text-gray-300 bg-gray-800/50 hover:bg-gray-800 border border-gray-700 rounded transition-colors"
          >
            Restart Session
          </button>
        ) : (
          <div className="space-y-1.5">
            <p className="text-[10px] text-amber-400">
              This will create a final checkpoint and start a new session with a compiled briefing.
            </p>
            <div className="flex gap-2">
              <button
                onClick={handleRestart}
                disabled={restarting}
                className="flex-1 py-1.5 text-xs font-medium text-white bg-amber-600 hover:bg-amber-500 disabled:bg-gray-700 disabled:text-gray-500 rounded transition-colors"
              >
                {restarting ? 'Restarting...' : 'Confirm Restart'}
              </button>
              <button
                onClick={() => setConfirmRestart(false)}
                disabled={restarting}
                className="px-3 py-1.5 text-xs font-medium text-gray-400 hover:text-gray-300 bg-gray-800 rounded transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}

// ============================================================================
// Session History Row (G5.3)
// ============================================================================

interface SessionRowProps {
  session: SessionRecord
}

function SessionRow({ session }: SessionRowProps) {
  const isCurrent = !session.ended_at
  const duration = session.ended_at
    ? Math.floor((session.ended_at - session.started_at) / 60000)
    : Math.floor((Date.now() - session.started_at) / 60000)

  return (
    <div className={`flex items-center justify-between px-2 py-1.5 rounded text-xs ${
      isCurrent ? 'bg-blue-500/10 border border-blue-500/20' : 'bg-gray-800/30'
    }`}>
      <div className="flex items-center gap-2">
        {isCurrent && <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" />}
        <span className="font-mono text-gray-400">{session.session_id.slice(0, 8)}</span>
        {session.reason && (
          <span className="text-[10px] text-gray-600">{session.reason}</span>
        )}
      </div>
      <div className="flex items-center gap-2">
        <span className="px-1.5 py-0.5 text-[10px] font-medium text-gray-500 bg-gray-800 rounded">
          {session.stage}
        </span>
        <span className="text-[10px] text-gray-600 font-mono">
          {formatDuration(duration)}
        </span>
      </div>
    </div>
  )
}

// ============================================================================
// Original Components
// ============================================================================

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
          {checkpoint.stage}
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

// ============================================================================
// Formatters
// ============================================================================

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

function formatDuration(minutes: number): string {
  if (minutes < 1) return '<1m'
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (mins === 0) return `${hours}h`
  return `${hours}h ${mins}m`
}
