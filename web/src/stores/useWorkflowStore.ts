import { create } from 'zustand'
import type {
  Phase,
  PhaseInfo,
  Task,
  TaskStatus,
  Gate,
  CheckpointSummary,
  PhasesResponse,
  TasksResponse,
  GateApprovalResponse,
  V4Event
} from '../types/v4'

interface WorkflowState {
  // State
  currentPhase: Phase
  phases: PhaseInfo[]
  tasks: Task[]
  gates: Record<string, Gate>
  checkpoints: CheckpointSummary[]
  loading: boolean
  error: string | null

  // Actions
  setPhases: (current: Phase, phases: PhaseInfo[]) => void
  setTasks: (tasks: Task[]) => void
  addTask: (task: Task) => void
  updateTask: (id: string, updates: Partial<Task>) => void
  setGate: (phase: Phase, gate: Gate) => void
  setCheckpoints: (checkpoints: CheckpointSummary[]) => void
  addCheckpoint: (checkpoint: CheckpointSummary) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void

  // Handle WebSocket events
  handleEvent: (event: V4Event) => void
}

export const useWorkflowStore = create<WorkflowState>()((set, get) => ({
  // Initial state
  currentPhase: 'idea',
  phases: [],
  tasks: [],
  gates: {},
  checkpoints: [],
  loading: false,
  error: null,

  // Actions
  setPhases: (current, phases) => set({ currentPhase: current, phases }),

  setTasks: (tasks) => set({ tasks }),

  addTask: (task) => set((state) => ({
    tasks: [...state.tasks, task]
  })),

  updateTask: (id, updates) => set((state) => ({
    tasks: state.tasks.map((t) =>
      t.id === id ? { ...t, ...updates } : t
    )
  })),

  setGate: (phase, gate) => set((state) => ({
    gates: { ...state.gates, [phase]: gate }
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
          currentPhase: event.state.current_phase,
          phases: event.state.phases,
          tasks: event.state.tasks,
          checkpoints: event.state.checkpoints
        })
        break

      case 'phase_changed':
        set((state) => ({
          currentPhase: event.phase,
          phases: state.phases.map((p) => ({
            ...p,
            status: p.phase === event.phase ? 'current' as const :
                    p.phase === event.previous ? 'complete' as const :
                    p.status
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
            [event.phase]: {
              ...state.gates[event.phase],
              status: event.status,
              criteria: event.criteria || state.gates[event.phase]?.criteria || []
            }
          }
        }))
        break

      case 'checkpoint_created':
        get().addCheckpoint({
          id: event.checkpoint_id,
          phase: event.phase,
          created_at: Date.now()
        })
        break
    }
  }
}))

// API functions
const API_BASE = '/api'

export async function fetchPhases(): Promise<PhasesResponse> {
  const res = await fetch(`${API_BASE}/phases`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function fetchTasks(filters?: {
  phase?: Phase
  zone?: string
  status?: TaskStatus
  persona?: string
}): Promise<Task[]> {
  const params = new URLSearchParams()
  if (filters?.phase) params.set('phase', filters.phase)
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
  phase?: Phase
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

export async function fetchGate(phase: Phase): Promise<Gate> {
  const res = await fetch(`${API_BASE}/gates/gate-${phase}`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function approveGate(
  phase: Phase,
  approvedBy: string
): Promise<GateApprovalResponse> {
  const res = await fetch(`${API_BASE}/gates/gate-${phase}/approve`, {
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
export const useCurrentPhase = () => useWorkflowStore((s) => s.currentPhase)
export const usePhases = () => useWorkflowStore((s) => s.phases)
export const useTasks = () => useWorkflowStore((s) => s.tasks)
export const useTasksForPhase = (phase: Phase) => {
  const tasks = useWorkflowStore((s) => s.tasks)
  return tasks.filter((t) => t.phase === phase)
}
export const useGate = (phase: Phase) => useWorkflowStore((s) => s.gates[phase])
export const useCheckpoints = () => useWorkflowStore((s) => s.checkpoints)
export const useWorkflowLoading = () => useWorkflowStore((s) => s.loading)
export const useWorkflowError = () => useWorkflowStore((s) => s.error)
