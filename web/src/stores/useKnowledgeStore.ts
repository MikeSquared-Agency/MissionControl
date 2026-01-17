import { create } from 'zustand'
import type {
  TokenBudget,
  BudgetStatus,
  CheckpointSummary,
  Checkpoint,
  Finding,
  Handoff,
  HandoffResponse,
  V4Event
} from '../types/v4'

interface KnowledgeState {
  // State
  budgets: Record<string, TokenBudget>
  checkpoints: CheckpointSummary[]
  findings: Finding[]
  recentHandoffs: Handoff[]
  loading: boolean
  error: string | null

  // Actions
  setBudget: (workerID: string, budget: TokenBudget) => void
  removeBudget: (workerID: string) => void
  setCheckpoints: (checkpoints: CheckpointSummary[]) => void
  addCheckpoint: (checkpoint: CheckpointSummary) => void
  addFinding: (finding: Finding) => void
  addHandoff: (handoff: Handoff) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void

  // Handle WebSocket events
  handleEvent: (event: V4Event) => void
}

export const useKnowledgeStore = create<KnowledgeState>()((set, get) => ({
  // Initial state
  budgets: {},
  checkpoints: [],
  findings: [],
  recentHandoffs: [],
  loading: false,
  error: null,

  // Actions
  setBudget: (workerID, budget) => set((state) => ({
    budgets: { ...state.budgets, [workerID]: budget }
  })),

  removeBudget: (workerID) => set((state) => {
    const { [workerID]: _, ...rest } = state.budgets
    return { budgets: rest }
  }),

  setCheckpoints: (checkpoints) => set({ checkpoints }),

  addCheckpoint: (checkpoint) => set((state) => ({
    checkpoints: [...state.checkpoints, checkpoint]
  })),

  addFinding: (finding) => set((state) => ({
    findings: [...state.findings, finding]
  })),

  addHandoff: (handoff) => set((state) => ({
    recentHandoffs: [handoff, ...state.recentHandoffs].slice(0, 50) // Keep last 50
  })),

  setLoading: (loading) => set({ loading }),

  setError: (error) => set({ error }),

  // Handle WebSocket events
  handleEvent: (event) => {
    switch (event.type) {
      case 'token_warning':
      case 'token_critical':
        get().setBudget(event.worker_id, {
          worker_id: event.worker_id,
          budget: event.budget,
          used: event.usage,
          status: event.status,
          remaining: event.remaining
        })
        break

      case 'checkpoint_created':
        get().addCheckpoint({
          id: event.checkpoint_id,
          phase: event.phase,
          created_at: Date.now()
        })
        break

      case 'handoff_received':
        // We only get partial info from event, store what we have
        get().addHandoff({
          task_id: event.task_id,
          worker_id: event.worker_id,
          status: event.status,
          findings: [],
          artifacts: [],
          timestamp: Date.now()
        })
        break
    }
  }
}))

// API functions
const API_BASE = '/api'

export async function fetchBudget(workerID: string): Promise<TokenBudget> {
  const res = await fetch(`${API_BASE}/budgets/${workerID}`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function createBudget(workerID: string, budget: number): Promise<TokenBudget> {
  const res = await fetch(`${API_BASE}/budgets/${workerID}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ budget })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function recordUsage(workerID: string, tokens: number): Promise<TokenBudget> {
  const res = await fetch(`${API_BASE}/budgets/${workerID}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tokens })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function submitHandoff(handoff: {
  task_id: string
  worker_id: string
  status: 'complete' | 'blocked' | 'partial'
  findings: Finding[]
  artifacts: string[]
  open_questions?: string[]
  blocked_reason?: string
}): Promise<HandoffResponse> {
  const res = await fetch(`${API_BASE}/handoffs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(handoff)
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

export async function fetchCheckpoint(id: string): Promise<Checkpoint> {
  const res = await fetch(`${API_BASE}/checkpoints/${id}`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function createCheckpoint(): Promise<CheckpointSummary> {
  const res = await fetch(`${API_BASE}/checkpoints`, {
    method: 'POST'
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

// Selectors
export const useBudgets = () => useKnowledgeStore((s) => s.budgets)
export const useBudget = (workerID: string) => useKnowledgeStore((s) => s.budgets[workerID])
export const useCheckpoints = () => useKnowledgeStore((s) => s.checkpoints)
export const useFindings = () => useKnowledgeStore((s) => s.findings)
export const useRecentHandoffs = () => useKnowledgeStore((s) => s.recentHandoffs)

// Computed selectors
export const useBudgetAlerts = () => {
  const budgets = useKnowledgeStore((s) => s.budgets)
  return Object.values(budgets).filter(
    (b) => b.status === 'warning' || b.status === 'critical' || b.status === 'exceeded'
  )
}

export const useTotalTokenUsage = () => {
  const budgets = useKnowledgeStore((s) => s.budgets)
  const values = Object.values(budgets)
  return {
    used: values.reduce((sum, b) => sum + b.used, 0),
    budget: values.reduce((sum, b) => sum + b.budget, 0),
    workers: values.length
  }
}
