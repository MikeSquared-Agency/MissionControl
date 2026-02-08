import { create } from 'zustand'
import type {
  Stage,
  StageInfo,
  Task,
  TaskStatus,
  Gate,
  CheckpointSummary,
  StagesResponse,
  TasksResponse,
  GateApprovalResponse,
  WorkflowEvent
} from '../types/workflow'

interface WorkflowState {
  // State
  currentStage: Stage
  stages: StageInfo[]
  tasks: Task[]
  gates: Record<string, Gate>
  checkpoints: CheckpointSummary[]
  loading: boolean
  error: string | null

  // Actions
  setStages: (current: Stage, stages: StageInfo[]) => void
  setTasks: (tasks: Task[]) => void
  addTask: (task: Task) => void
  updateTask: (id: string, updates: Partial<Task>) => void
  setGate: (stage: Stage, gate: Gate) => void
  setCheckpoints: (checkpoints: CheckpointSummary[]) => void
  addCheckpoint: (checkpoint: CheckpointSummary) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void

  // Handle WebSocket events
  handleEvent: (event: WorkflowEvent) => void
}

export const useWorkflowStore = create<WorkflowState>()((set, get) => ({
  // Initial state
  currentStage: 'discovery',
  stages: [],
  tasks: [],
  gates: {},
  checkpoints: [],
  loading: false,
  error: null,

  // Actions
  setStages: (current, stages) => set({ currentStage: current, stages }),

  setTasks: (tasks) => set({ tasks }),

  addTask: (task) => set((state) => ({
    tasks: [...state.tasks, task]
  })),

  updateTask: (id, updates) => set((state) => ({
    tasks: state.tasks.map((t) =>
      t.id === id ? { ...t, ...updates } : t
    )
  })),

  setGate: (stage, gate) => set((state) => ({
    gates: { ...state.gates, [stage]: gate }
  })),

  setCheckpoints: (checkpoints) => set({ checkpoints }),

  addCheckpoint: (checkpoint) => set((state) => ({
    checkpoints: [...state.checkpoints, checkpoint]
  })),

  setLoading: (loading) => set({ loading }),

  setError: (error) => set({ error }),

  // Handle WebSocket events
  handleEvent: (event) => {
    switch (event.type) {
      case 'v4_state':
        set({
          currentStage: event.state.current_stage,
          stages: event.state.stages,
          tasks: event.state.tasks,
          checkpoints: event.state.checkpoints
        })
        break

      case 'stage_changed':
        set((state) => ({
          currentStage: event.stage,
          stages: state.stages.map((s) => ({
            ...s,
            status: s.stage === event.stage ? 'current' as const :
                    s.stage === event.previous ? 'complete' as const :
                    s.status
          }))
        }))
        break

      case 'task_created':
        get().addTask(event.task)
        break

      case 'task_updated':
        get().updateTask(event.task_id, { status: event.status })
        break

      case 'gate_status':
        set((state) => ({
          gates: {
            ...state.gates,
            [event.stage]: {
              ...state.gates[event.stage],
              status: event.status,
              criteria: event.criteria || state.gates[event.stage]?.criteria || []
            }
          }
        }))
        break

      case 'checkpoint_created':
        get().addCheckpoint({
          id: event.checkpoint_id,
          stage: event.stage,
          created_at: Date.now()
        })
        break
    }
  }
}))

// API functions
const API_BASE = '/api'

export async function fetchStages(): Promise<StagesResponse> {
  const res = await fetch(`${API_BASE}/stages`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function fetchTasks(filters?: {
  stage?: Stage
  zone?: string
  status?: TaskStatus
  persona?: string
}): Promise<Task[]> {
  const params = new URLSearchParams()
  if (filters?.stage) params.set('stage', filters.stage)
  if (filters?.zone) params.set('zone', filters.zone)
  if (filters?.status) params.set('status', filters.status)
  if (filters?.persona) params.set('persona', filters.persona)

  const url = params.toString() ? `${API_BASE}/tasks?${params}` : `${API_BASE}/tasks`
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const data: TasksResponse = await res.json()
  return data.tasks || []
}

export async function createTask(task: {
  name: string
  stage?: Stage
  zone: string
  persona: string
  dependencies?: string[]
}): Promise<Task> {
  const res = await fetch(`${API_BASE}/tasks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(task)
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function updateTaskStatus(
  id: string,
  status: TaskStatus,
  reason?: string
): Promise<Task> {
  const res = await fetch(`${API_BASE}/tasks/${id}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status, reason })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function fetchGate(stage: Stage): Promise<Gate> {
  const res = await fetch(`${API_BASE}/gates/gate-${stage}`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function approveGate(
  stage: Stage,
  approvedBy: string
): Promise<GateApprovalResponse> {
  const res = await fetch(`${API_BASE}/gates/gate-${stage}/approve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ approved_by: approvedBy })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function fetchCheckpoints(): Promise<CheckpointSummary[]> {
  const res = await fetch(`${API_BASE}/checkpoints`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const data = await res.json()
  return data.checkpoints || []
}

// Selectors
export const useCurrentStage = () => useWorkflowStore((s) => s.currentStage)
export const useStages = () => useWorkflowStore((s) => s.stages)
export const useTasks = () => useWorkflowStore((s) => s.tasks)
export const useTasksForStage = (stage: Stage) => {
  const tasks = useWorkflowStore((s) => s.tasks)
  return tasks.filter((t) => t.stage === stage)
}
export const useGate = (stage: Stage) => useWorkflowStore((s) => s.gates[stage])
export const useCheckpoints = () => useWorkflowStore((s) => s.checkpoints)
export const useWorkflowLoading = () => useWorkflowStore((s) => s.loading)
export const useWorkflowError = () => useWorkflowStore((s) => s.error)
