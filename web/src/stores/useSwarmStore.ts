import { create } from 'zustand'
import type {
  SwarmOverview,
  WarrenSSEEvent,
  SwarmAlert
} from '../types/swarm'

const MAX_EVENTS = 100
const MAX_ALERTS = 50

interface SwarmState {
  // State
  overview: SwarmOverview | null
  events: WarrenSSEEvent[]
  alerts: SwarmAlert[]
  loading: boolean
  error: string | null
  lastFetched: number | null

  // Actions
  setOverview: (data: SwarmOverview) => void
  addEvent: (event: WarrenSSEEvent) => void
  addAlert: (alert: SwarmAlert) => void
  dismissAlert: (id: string) => void
  setLoading: (loading: boolean) => void
  setError: (error: string | null) => void
}

export const useSwarmStore = create<SwarmState>()((set, get) => ({
  overview: null,
  events: [],
  alerts: [],
  loading: false,
  error: null,
  lastFetched: null,

  setOverview: (data) => {
    const prev = get().overview
    const newAlerts: SwarmAlert[] = []

    // Derive alerts from errors map
    for (const [service, msg] of Object.entries(data.errors)) {
      // Only alert if this is a new error (wasn't in previous)
      if (!prev?.errors[service]) {
        newAlerts.push({
          id: `down-${service}-${Date.now()}`,
          level: 'critical',
          service,
          message: `${service} is unreachable: ${msg}`,
          timestamp: Date.now()
        })
      }
    }

    // DLQ spike detection
    const dlqDepth = data.chronicle?.dlq?.depth ?? 0
    const prevDlqDepth = prev?.chronicle?.dlq?.depth ?? 0
    if (dlqDepth > 10 && dlqDepth > prevDlqDepth * 2) {
      newAlerts.push({
        id: `dlq-spike-${Date.now()}`,
        level: 'warning',
        service: 'chronicle',
        message: `DLQ depth spiked to ${dlqDepth}`,
        timestamp: Date.now()
      })
    }

    set((state) => ({
      overview: data,
      loading: false,
      error: null,
      lastFetched: Date.now(),
      alerts: [...newAlerts, ...state.alerts].slice(0, MAX_ALERTS)
    }))
  },

  addEvent: (event) => set((state) => ({
    events: [event, ...state.events].slice(0, MAX_EVENTS)
  })),

  addAlert: (alert) => set((state) => ({
    alerts: [alert, ...state.alerts].slice(0, MAX_ALERTS)
  })),

  dismissAlert: (id) => set((state) => ({
    alerts: state.alerts.filter((a) => a.id !== id)
  })),

  setLoading: (loading) => set({ loading }),

  setError: (error) => set({ error, loading: false })
}))

// Selectors
export const useSwarmOverview = () => useSwarmStore((s) => s.overview)
export const useSwarmEvents = () => useSwarmStore((s) => s.events)
export const useSwarmAlerts = () => useSwarmStore((s) => s.alerts)
export const useSwarmLoading = () => useSwarmStore((s) => s.loading)

// Computed selectors
export function useFleetSummary() {
  const overview = useSwarmStore((s) => s.overview)
  if (!overview) return null

  const warrenAgents = overview.warren?.agents ?? []
  const dispatchAgents = overview.dispatch?.agents ?? []

  const totalAgents = warrenAgents.length + dispatchAgents.length
  const readyCount = warrenAgents.filter((a) => a.state === 'ready').length
  const sleepingCount = warrenAgents.filter((a) => a.state === 'sleeping').length
  const degradedCount = Object.keys(overview.errors).length

  return {
    totalAgents,
    readyCount,
    sleepingCount,
    degradedCount,
    activeTasks: overview.dispatch?.stats?.in_progress ?? 0,
    dlqDepth: overview.chronicle?.dlq?.depth ?? 0,
    promptCount: overview.promptforge?.prompt_count ?? 0,
    collectionCount: overview.alexandria?.collection_count ?? 0
  }
}

export function usePipelineSummary() {
  const overview = useSwarmStore((s) => s.overview)
  if (!overview?.dispatch?.stats) return null

  const stats = overview.dispatch.stats
  return {
    pending: stats.pending ?? 0,
    inProgress: stats.in_progress ?? 0,
    completed: stats.completed ?? 0,
    failed: stats.failed ?? 0,
    total: stats.total ?? 0,
    dlqDepth: overview.chronicle?.dlq?.depth ?? 0
  }
}

// API
const API_BASE = '/api'

export async function fetchSwarmOverview(): Promise<SwarmOverview> {
  const res = await fetch(`${API_BASE}/swarm/overview`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}
