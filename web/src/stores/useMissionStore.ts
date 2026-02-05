import { create } from 'zustand'
import type { KingQuestion } from '../types'

export interface MissionWorker {
  id: string
  persona: string
  taskId: string
  zone: string
  status: 'starting' | 'running' | 'completed' | 'error'
  pid?: number
  startedAt: string
}

export interface MissionTask {
  id: string
  name: string
  stage: string
  zone: string
  persona: string
  status: string
  workerId?: string
  createdAt: string
  updatedAt: string
}

export interface MissionGate {
  stage: string
  status: 'pending' | 'awaiting_approval' | 'approved'
  criteria: string[]
  approvedAt?: string
}

export interface Finding {
  taskId: string
  workerId: string
  findingType: 'discovery' | 'blocker' | 'decision' | 'concern'
  summary: string
  detailsPath?: string
  severity?: string
}

export interface MissionState {
  // State
  initialized: boolean
  currentStage: string
  workers: MissionWorker[]
  tasks: MissionTask[]
  gates: Record<string, MissionGate>
  findings: Finding[]
  kingRunning: boolean
  kingConnected: boolean
  kingQuestion: KingQuestion | null

  // Actions
  setInitialized: (initialized: boolean) => void
  setCurrentStage: (stage: string) => void
  setWorkers: (workers: MissionWorker[]) => void
  addWorker: (worker: MissionWorker) => void
  updateWorker: (id: string, updates: Partial<MissionWorker>) => void
  removeWorker: (id: string) => void
  setTasks: (tasks: MissionTask[]) => void
  addTask: (task: MissionTask) => void
  updateTask: (id: string, updates: Partial<MissionTask>) => void
  setGate: (stage: string, gate: MissionGate) => void
  addFinding: (finding: Finding) => void
  setFindings: (findings: Finding[]) => void
  setKingRunning: (running: boolean) => void
  setKingConnected: (connected: boolean) => void
  setKingQuestion: (question: KingQuestion | null) => void

  // Handle WebSocket events
  handleEvent: (event: V5Event) => void
}

export type V5Event =
  | { type: 'mission_state'; state: { stage: string; workers: MissionWorker[]; tasks: MissionTask[]; gates: Record<string, MissionGate> } }
  | { type: 'stage_changed'; stage: string }
  | { type: 'task_created'; task: MissionTask }
  | { type: 'task_updated'; task_id: string; status: string }
  | { type: 'worker_spawned'; worker_id: string; persona: string; task_id: string; zone: string }
  | { type: 'worker_completed'; worker_id: string }
  | { type: 'findings_ready'; task_id: string }
  | { type: 'gate_ready'; stage: string }
  | { type: 'gate_approved'; stage: string }
  | { type: 'king_output'; data: unknown }
  | { type: 'king_message'; data: { role: string; content: string; timestamp: number } }
  | { type: 'king_status'; is_running: boolean }
  | { type: 'king_started'; started_at: string }
  | { type: 'king_stopped' }
  | { type: 'king_question'; data: KingQuestion }

export const useMissionStore = create<MissionState>()((set, get) => ({
  // Initial state
  initialized: false,
  currentStage: 'discovery',
  workers: [],
  tasks: [],
  gates: {},
  findings: [],
  kingRunning: false,
  kingConnected: false,
  kingQuestion: null,

  // Actions
  setInitialized: (initialized) => set({ initialized }),
  setCurrentStage: (stage) => set({ currentStage: stage }),

  setWorkers: (workers) => set({ workers }),

  addWorker: (worker) => set((state) => ({
    workers: [...state.workers, worker]
  })),

  updateWorker: (id, updates) => set((state) => ({
    workers: state.workers.map((w) =>
      w.id === id ? { ...w, ...updates } : w
    )
  })),

  removeWorker: (id) => set((state) => ({
    workers: state.workers.filter((w) => w.id !== id)
  })),

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

  addFinding: (finding) => set((state) => ({
    findings: [...state.findings, finding]
  })),

  setFindings: (findings) => set({ findings }),

  setKingRunning: (running) => set({ kingRunning: running }),
  setKingConnected: (connected) => set({ kingConnected: connected }),
  setKingQuestion: (question) => set({ kingQuestion: question }),

  // Handle WebSocket events
  handleEvent: (event) => {
    switch (event.type) {
      case 'mission_state':
        set({
          initialized: true,
          currentStage: event.state.stage,
          workers: event.state.workers,
          tasks: event.state.tasks,
          gates: event.state.gates
        })
        break

      case 'stage_changed':
        set({ currentStage: event.stage })
        break

      case 'task_created':
        get().addTask(event.task)
        break

      case 'task_updated':
        get().updateTask(event.task_id, { status: event.status })
        break

      case 'worker_spawned':
        get().addWorker({
          id: event.worker_id,
          persona: event.persona,
          taskId: event.task_id,
          zone: event.zone,
          status: 'starting',
          startedAt: new Date().toISOString()
        })
        break

      case 'worker_completed':
        get().updateWorker(event.worker_id, { status: 'completed' })
        break

      case 'gate_ready':
        set((state) => ({
          gates: {
            ...state.gates,
            [event.stage]: {
              ...state.gates[event.stage],
              status: 'awaiting_approval'
            }
          }
        }))
        break

      case 'gate_approved':
        set((state) => ({
          gates: {
            ...state.gates,
            [event.stage]: {
              ...state.gates[event.stage],
              status: 'approved',
              approvedAt: new Date().toISOString()
            }
          }
        }))
        break

      case 'king_status':
        set({ kingRunning: event.is_running })
        break

      case 'king_started':
        set({ kingRunning: true })
        break

      case 'king_stopped':
        set({ kingRunning: false, kingQuestion: null })
        break

      case 'king_output':
        // King output is handled by the main store's king conversation
        break

      case 'king_question':
        // Claude is asking a question - show question UI
        set({ kingQuestion: event.data })
        break

      case 'findings_ready':
        // Trigger findings refresh
        console.log('Findings ready for task:', event.task_id)
        break
    }
  }
}))

// Selectors
export const useCurrentStage = () => useMissionStore((s) => s.currentStage)
export const useMissionWorkers = () => useMissionStore((s) => s.workers)
export const useMissionTasks = () => useMissionStore((s) => s.tasks)
export const useMissionGates = () => useMissionStore((s) => s.gates)
export const useFindings = () => useMissionStore((s) => s.findings)
export const useKingRunning = () => useMissionStore((s) => s.kingRunning)
export const useKingConnected = () => useMissionStore((s) => s.kingConnected)
export const useKingQuestion = () => useMissionStore((s) => s.kingQuestion)
export const useMissionInitialized = () => useMissionStore((s) => s.initialized)

// API functions
const API_BASE = '/api'

export async function fetchMissionState(): Promise<void> {
  const res = await fetch(`${API_BASE}/mission/state`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const data = await res.json()
  useMissionStore.getState().handleEvent({ type: 'mission_state', state: data })
}

export async function startKing(): Promise<void> {
  const res = await fetch(`${API_BASE}/king/start`, { method: 'POST' })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export async function stopKing(): Promise<void> {
  const res = await fetch(`${API_BASE}/king/stop`, { method: 'POST' })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export async function sendKingMessage(message: string): Promise<void> {
  const res = await fetch(`${API_BASE}/king/message`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content: message })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export async function answerKingQuestion(optionIndex: number): Promise<void> {
  const res = await fetch(`${API_BASE}/king/answer`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ option_index: optionIndex })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  // Clear the question from state after answering
  useMissionStore.getState().setKingQuestion(null)
}

export async function approveGate(stage: string): Promise<void> {
  const res = await fetch(`${API_BASE}/mission/gates/${stage}/approve`, {
    method: 'POST'
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export async function fetchFindings(taskId?: string): Promise<Finding[]> {
  const url = taskId
    ? `${API_BASE}/mission/findings?task_id=${taskId}`
    : `${API_BASE}/mission/findings`
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const data = await res.json()
  return data.findings || []
}
