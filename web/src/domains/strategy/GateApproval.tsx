import { useEffect, useState } from 'react'
import {
  useWorkflowStore,
  useCurrentPhase,
  usePhases,
  fetchGate,
  approveGate
} from '../../stores/useWorkflowStore'
import type { Phase, Gate, GateCriterion } from '../../types/v4'
import {
  ALL_PHASES,
  getPhaseLabel,
  getGateStatusColor,
  getNextPhase
} from '../../types/v4'

export function GateApproval() {
  const currentPhase = useCurrentPhase()
  const phases = usePhases()
  const gates = useWorkflowStore((s) => s.gates)
  const setGate = useWorkflowStore((s) => s.setGate)
  const [loading, setLoading] = useState(true)

  // Load all gates on mount
  useEffect(() => {
    async function loadGates() {
      setLoading(true)
      try {
        await Promise.all(
          ALL_PHASES.map(async (phase) => {
            try {
              const gate = await fetchGate(phase)
              setGate(phase, gate)
            } catch (err) {
              // Gate might not exist yet, that's ok
            }
          })
        )
      } finally {
        setLoading(false)
      }
    }
    loadGates()
  }, [setGate])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500 text-sm">
        Loading gates...
      </div>
    )
  }

  // Find gates that need attention
  const awaitingApproval = ALL_PHASES.filter(
    (p) => gates[p]?.status === 'awaiting_approval'
  )

  return (
    <div className="h-full flex flex-col bg-gray-900">
      {/* Header */}
      <div className="p-3 bg-gray-850 border-b border-gray-800">
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-sm font-semibold text-gray-200">Gate Approval</h2>
          <span className="text-xs text-gray-500">
            Current: {getPhaseLabel(currentPhase)}
          </span>
        </div>

        {/* Alerts */}
        {awaitingApproval.length > 0 && (
          <div className="flex items-center gap-2 px-2 py-1.5 rounded bg-amber-500/10 border border-amber-500/20">
            <span className="w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
            <span className="text-xs text-amber-400">
              {awaitingApproval.length} gate{awaitingApproval.length !== 1 ? 's' : ''} awaiting approval
            </span>
          </div>
        )}
      </div>

      {/* Gates list */}
      <div className="flex-1 overflow-auto p-3">
        <div className="space-y-3">
          {ALL_PHASES.map((phase, index) => {
            const gate = gates[phase]
            const phaseInfo = phases.find((p) => p.phase === phase)
            const isCurrentPhase = phase === currentPhase
            const isPastPhase = phaseInfo?.status === 'complete'
            const isFuturePhase = !isCurrentPhase && !isPastPhase

            return (
              <GateCard
                key={phase}
                phase={phase}
                gate={gate}
                isCurrentPhase={isCurrentPhase}
                isPastPhase={isPastPhase}
                isFuturePhase={isFuturePhase}
                phaseNumber={index + 1}
              />
            )
          })}
        </div>
      </div>
    </div>
  )
}

interface GateCardProps {
  phase: Phase
  gate: Gate | undefined
  isCurrentPhase: boolean
  isPastPhase: boolean
  isFuturePhase: boolean
  phaseNumber: number
}

function GateCard({
  phase,
  gate,
  isCurrentPhase,
  isPastPhase,
  isFuturePhase,
  phaseNumber
}: GateCardProps) {
  const [approving, setApproving] = useState(false)
  const [expanded, setExpanded] = useState(isCurrentPhase)
  const setGate = useWorkflowStore((s) => s.setGate)

  const handleApprove = async () => {
    setApproving(true)
    try {
      const result = await approveGate(phase, 'user')
      setGate(phase, result.gate)
    } catch (err) {
      console.error('Failed to approve gate:', err)
    } finally {
      setApproving(false)
    }
  }

  const status = gate?.status || 'closed'
  const criteria = gate?.criteria || []
  const allCriteriaMet = criteria.every((c) => c.satisfied)
  const satisfiedCount = criteria.filter((c) => c.satisfied).length
  const nextPhase = getNextPhase(phase)

  return (
    <div
      className={`
        rounded-lg border transition-all
        ${isCurrentPhase
          ? 'bg-blue-500/5 border-blue-500/30'
          : isPastPhase
            ? 'bg-green-500/5 border-green-500/20'
            : 'bg-gray-800/30 border-gray-800'
        }
        ${status === 'awaiting_approval' ? 'ring-1 ring-amber-500/50' : ''}
      `}
    >
      {/* Header - always visible */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full p-3 flex items-center gap-3 text-left"
      >
        {/* Phase number */}
        <div
          className={`
            w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold
            ${isPastPhase
              ? 'bg-green-500 text-white'
              : isCurrentPhase
                ? 'bg-blue-500 text-white'
                : 'bg-gray-700 text-gray-400'
            }
          `}
        >
          {isPastPhase ? '✓' : phaseNumber}
        </div>

        {/* Phase name and status */}
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <span className={`
              text-sm font-medium
              ${isPastPhase ? 'text-green-400' : isCurrentPhase ? 'text-blue-400' : 'text-gray-400'}
            `}>
              {getPhaseLabel(phase)}
            </span>
            {isCurrentPhase && (
              <span className="px-1.5 py-0.5 text-[10px] font-medium bg-blue-500/20 text-blue-400 rounded">
                CURRENT
              </span>
            )}
          </div>
          <div className="text-[10px] text-gray-500 mt-0.5">
            {satisfiedCount}/{criteria.length} criteria met
          </div>
        </div>

        {/* Status badge */}
        <span
          className="px-2 py-1 text-[10px] font-medium rounded"
          style={{
            backgroundColor: `${getGateStatusColor(status)}20`,
            color: getGateStatusColor(status)
          }}
        >
          {status === 'awaiting_approval' ? 'AWAITING' : status.toUpperCase()}
        </span>

        {/* Expand icon */}
        <svg
          className={`w-4 h-4 text-gray-500 transition-transform ${expanded ? 'rotate-180' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="px-3 pb-3 border-t border-gray-800/50">
          {/* Criteria checklist */}
          <div className="mt-3 space-y-2">
            <h4 className="text-[10px] font-medium text-gray-500 uppercase tracking-wider">
              Gate Criteria
            </h4>
            {criteria.length === 0 ? (
              <div className="text-xs text-gray-600 py-2">No criteria defined</div>
            ) : (
              <div className="space-y-1.5">
                {criteria.map((criterion, i) => (
                  <CriterionRow key={i} criterion={criterion} index={i} />
                ))}
              </div>
            )}
          </div>

          {/* Progress bar */}
          <div className="mt-3">
            <div className="h-1.5 bg-gray-800 rounded-full overflow-hidden">
              <div
                className="h-full rounded-full transition-all duration-300"
                style={{
                  width: `${criteria.length > 0 ? (satisfiedCount / criteria.length) * 100 : 0}%`,
                  backgroundColor: allCriteriaMet ? '#22c55e' : '#f59e0b'
                }}
              />
            </div>
          </div>

          {/* Action buttons */}
          <div className="mt-3 flex items-center gap-2">
            {status !== 'open' && (
              <button
                onClick={handleApprove}
                disabled={approving || isFuturePhase}
                className={`
                  flex-1 py-2 text-xs font-medium rounded transition-colors
                  ${allCriteriaMet || status === 'awaiting_approval'
                    ? 'bg-green-600 hover:bg-green-500 text-white'
                    : 'bg-amber-600 hover:bg-amber-500 text-white'
                  }
                  disabled:bg-gray-700 disabled:text-gray-500 disabled:cursor-not-allowed
                `}
              >
                {approving
                  ? 'Approving...'
                  : allCriteriaMet
                    ? 'Approve Gate'
                    : 'Override & Approve'
                }
              </button>
            )}

            {status === 'open' && nextPhase && (
              <div className="flex-1 py-2 text-xs text-center text-green-400">
                ✓ Gate approved — ready for {getPhaseLabel(nextPhase)}
              </div>
            )}

            {status === 'open' && !nextPhase && (
              <div className="flex-1 py-2 text-xs text-center text-green-400">
                ✓ Final gate approved — project complete!
              </div>
            )}
          </div>

          {/* Approval info */}
          {gate?.approved_by && (
            <div className="mt-2 text-[10px] text-gray-500 text-center">
              Approved by <span className="text-gray-400">{gate.approved_by}</span>
              {gate.approved_at && (
                <span> on {new Date(gate.approved_at * 1000).toLocaleDateString()}</span>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

interface CriterionRowProps {
  criterion: GateCriterion
  index: number
}

function CriterionRow({ criterion, index }: CriterionRowProps) {
  return (
    <div className="flex items-start gap-2">
      <div
        className={`
          mt-0.5 w-5 h-5 rounded flex items-center justify-center flex-shrink-0
          ${criterion.satisfied
            ? 'bg-green-500/20 text-green-500'
            : 'bg-gray-700 text-gray-500'
          }
        `}
      >
        {criterion.satisfied ? (
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path
              fillRule="evenodd"
              d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
              clipRule="evenodd"
            />
          </svg>
        ) : (
          <span className="text-[10px]">{index + 1}</span>
        )}
      </div>
      <span
        className={`
          text-xs leading-relaxed
          ${criterion.satisfied ? 'text-gray-300' : 'text-gray-500'}
        `}
      >
        {criterion.description}
      </span>
    </div>
  )
}
