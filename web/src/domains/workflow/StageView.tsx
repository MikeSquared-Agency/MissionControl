import { useEffect, useState } from 'react'
import {
  useWorkflowStore,
  useCurrentStage,
  useStages,
  useTasksForStage,
  useGate,
  fetchStages,
  fetchTasks,
  fetchGate,
  approveGate,
  updateTaskStatus
} from '../../stores/useWorkflowStore'
import type { Stage, Task, Gate, TaskStatus } from '../../types/workflow'
import {
  ALL_STAGES,
  getStageLabel,
  getTaskStatusColor,
  getGateStatusColor
} from '../../types/workflow'

export function StageView() {
  const currentStage = useCurrentStage()
  const stages = useStages()
  const setStages = useWorkflowStore((s) => s.setStages)
  const setTasks = useWorkflowStore((s) => s.setTasks)
  const setGate = useWorkflowStore((s) => s.setGate)
  const [loading, setLoading] = useState(true)
  const [selectedStage, setSelectedStage] = useState<Stage>(currentStage)

  // Load initial data
  useEffect(() => {
    async function load() {
      try {
        const [stagesData, tasksData] = await Promise.all([
          fetchStages(),
          fetchTasks()
        ])
        setStages(stagesData.current, stagesData.stages)
        setTasks(tasksData)
        setSelectedStage(stagesData.current)

        // Load gate for current stage
        const gate = await fetchGate(stagesData.current)
        setGate(stagesData.current, gate)
      } catch (err) {
        console.error('Failed to load workflow data:', err)
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [setStages, setTasks, setGate])

  // Load gate when selected stage changes
  useEffect(() => {
    async function loadGate() {
      try {
        const gate = await fetchGate(selectedStage)
        setGate(selectedStage, gate)
      } catch (err) {
        console.error('Failed to load gate:', err)
      }
    }
    loadGate()
  }, [selectedStage, setGate])

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full text-gray-500 text-sm">
        Loading workflow...
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col bg-gray-900">
      {/* Stage timeline */}
      <StageTimeline
        stages={stages.length > 0 ? stages : ALL_STAGES.map(s => ({
          stage: s,
          status: s === currentStage ? 'current' : 'pending'
        }))}
        currentStage={currentStage}
        selectedStage={selectedStage}
        onSelectStage={setSelectedStage}
      />

      {/* Selected stage content */}
      <div className="flex-1 overflow-auto p-4">
        <StageDetail stage={selectedStage} />
      </div>
    </div>
  )
}

interface StageTimelineProps {
  stages: Array<{ stage: Stage; status: string }>
  currentStage: Stage
  selectedStage: Stage
  onSelectStage: (stage: Stage) => void
}

function StageTimeline({ stages, currentStage: _currentStage, selectedStage, onSelectStage }: StageTimelineProps) {
  return (
    <div className="flex items-center gap-1 p-3 bg-gray-850 border-b border-gray-800">
      {stages.map((s, i) => (
        <div key={s.stage} className="flex items-center">
          {/* Stage node */}
          <button
            onClick={() => onSelectStage(s.stage)}
            className={`
              flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium
              transition-all
              ${s.stage === selectedStage
                ? 'bg-blue-600 text-white'
                : s.status === 'complete'
                  ? 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
                  : s.status === 'current'
                    ? 'bg-amber-500/20 text-amber-400 hover:bg-amber-500/30'
                    : 'bg-gray-800 text-gray-500 hover:bg-gray-700'
              }
            `}
          >
            {/* Status indicator */}
            <span className={`
              w-2 h-2 rounded-full
              ${s.status === 'complete' ? 'bg-green-500' :
                s.status === 'current' ? 'bg-amber-500 animate-pulse' :
                'bg-gray-600'}
            `} />
            {getStageLabel(s.stage)}
          </button>

          {/* Connector line */}
          {i < stages.length - 1 && (
            <div className={`
              w-6 h-0.5 mx-1
              ${s.status === 'complete' ? 'bg-green-500/50' : 'bg-gray-700'}
            `} />
          )}
        </div>
      ))}
    </div>
  )
}

interface StageDetailProps {
  stage: Stage
}

function StageDetail({ stage }: StageDetailProps) {
  const tasks = useTasksForStage(stage)
  const gate = useGate(stage)

  return (
    <div className="space-y-4">
      {/* Stage header */}
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-200">
          {getStageLabel(stage)} Stage
        </h2>
        <span className="text-xs text-gray-500">
          {tasks.length} task{tasks.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Gate status */}
      {gate && <GateCard gate={gate} stage={stage} />}

      {/* Tasks list */}
      <div className="space-y-2">
        <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wider">
          Tasks
        </h3>
        {tasks.length === 0 ? (
          <div className="text-sm text-gray-600 py-4 text-center">
            No tasks in this stage
          </div>
        ) : (
          <div className="space-y-1">
            {tasks.map((task) => (
              <TaskCard key={task.id} task={task} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

interface GateCardProps {
  gate: Gate
  stage: Stage
}

function GateCard({ gate, stage }: GateCardProps) {
  const [approving, setApproving] = useState(false)
  const setGate = useWorkflowStore((s) => s.setGate)

  const handleApprove = async () => {
    setApproving(true)
    try {
      const result = await approveGate(stage, 'user')
      setGate(stage, result.gate)
    } catch (err) {
      console.error('Failed to approve gate:', err)
    } finally {
      setApproving(false)
    }
  }

  const allCriteriaMet = gate.criteria.every((c) => c.satisfied)

  return (
    <div className="p-3 rounded-lg bg-gray-800/50 border border-gray-700">
      {/* Gate header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span
            className="w-2.5 h-2.5 rounded-full"
            style={{ backgroundColor: getGateStatusColor(gate.status) }}
          />
          <span className="text-sm font-medium text-gray-300">
            Stage Gate
          </span>
        </div>
        <span
          className="px-2 py-0.5 text-xs font-medium rounded"
          style={{
            backgroundColor: `${getGateStatusColor(gate.status)}20`,
            color: getGateStatusColor(gate.status)
          }}
        >
          {gate.status.replace('_', ' ')}
        </span>
      </div>

      {/* Criteria checklist */}
      <div className="space-y-1.5 mb-3">
        {gate.criteria.map((criterion, i) => (
          <div key={i} className="flex items-center gap-2 text-xs">
            <span className={`
              w-4 h-4 rounded flex items-center justify-center
              ${criterion.satisfied
                ? 'bg-green-500/20 text-green-500'
                : 'bg-gray-700 text-gray-500'
              }
            `}>
              {criterion.satisfied ? '\u2713' : '\u25CB'}
            </span>
            <span className={criterion.satisfied ? 'text-gray-300' : 'text-gray-500'}>
              {criterion.description}
            </span>
          </div>
        ))}
      </div>

      {/* Approve button */}
      {gate.status !== 'open' && (
        <button
          onClick={handleApprove}
          disabled={approving || (!allCriteriaMet && gate.status !== 'awaiting_approval')}
          className={`
            w-full py-1.5 text-xs font-medium rounded transition-colors
            ${allCriteriaMet || gate.status === 'awaiting_approval'
              ? 'bg-green-600 hover:bg-green-500 text-white'
              : 'bg-gray-700 text-gray-500 cursor-not-allowed'
            }
          `}
        >
          {approving ? 'Approving...' : 'Approve Gate'}
        </button>
      )}

      {/* Approval info */}
      {gate.approved_by && (
        <div className="mt-2 text-xs text-gray-500">
          Approved by {gate.approved_by}
          {gate.approved_at && ` on ${new Date(gate.approved_at * 1000).toLocaleDateString()}`}
        </div>
      )}
    </div>
  )
}

interface TaskCardProps {
  task: Task
}

function TaskCard({ task }: TaskCardProps) {
  const [updating, setUpdating] = useState(false)
  const updateTask = useWorkflowStore((s) => s.updateTask)

  const handleStatusChange = async (newStatus: TaskStatus) => {
    setUpdating(true)
    try {
      const updated = await updateTaskStatus(task.id, newStatus)
      updateTask(task.id, updated)
    } catch (err) {
      console.error('Failed to update task:', err)
    } finally {
      setUpdating(false)
    }
  }

  const statusOptions: TaskStatus[] = ['pending', 'ready', 'in_progress', 'blocked', 'done']

  return (
    <div className="p-2.5 rounded bg-gray-800/30 border border-gray-800 hover:border-gray-700 transition-colors">
      <div className="flex items-start gap-2">
        {/* Status indicator */}
        <span
          className="mt-0.5 w-2 h-2 rounded-full flex-shrink-0"
          style={{ backgroundColor: getTaskStatusColor(task.status) }}
        />

        {/* Task info */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-200 truncate">
              {task.name}
            </span>
            {task.persona && (
              <span className="px-1.5 py-0.5 text-[10px] font-medium text-gray-500 bg-gray-800 rounded">
                {task.persona}
              </span>
            )}
          </div>

          {/* Zone and dependencies */}
          <div className="mt-1 flex items-center gap-2 text-[10px] text-gray-500">
            {task.zone && <span>Zone: {task.zone}</span>}
            {task.dependencies.length > 0 && (
              <span>Deps: {task.dependencies.length}</span>
            )}
          </div>

          {/* Blocked reason */}
          {task.status === 'blocked' && task.blocked_reason && (
            <div className="mt-1 text-[10px] text-red-400">
              Blocked: {task.blocked_reason}
            </div>
          )}
        </div>

        {/* Status dropdown */}
        <select
          value={task.status}
          onChange={(e) => handleStatusChange(e.target.value as TaskStatus)}
          disabled={updating}
          className="px-1.5 py-0.5 text-[10px] font-medium bg-gray-800 border border-gray-700 rounded text-gray-400 focus:outline-none focus:border-gray-600"
        >
          {statusOptions.map((status) => (
            <option key={status} value={status}>
              {status.replace('_', ' ')}
            </option>
          ))}
        </select>
      </div>
    </div>
  )
}
