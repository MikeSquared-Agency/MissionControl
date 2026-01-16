import { create } from 'zustand'
import type { Agent, Event, SpawnRequest } from '../types'

interface AgentState {
  agents: Agent[]
  selectedAgentId: string | null
  events: Event[]

  // Actions
  setAgents: (agents: Agent[]) => void
  addAgent: (agent: Agent) => void
  updateAgent: (id: string, updates: Partial<Agent>) => void
  removeAgent: (id: string) => void
  selectAgent: (id: string | null) => void
  addEvent: (event: Event) => void
  clearEvents: () => void
}

export const useAgentStore = create<AgentState>((set) => ({
  agents: [],
  selectedAgentId: null,
  events: [],

  setAgents: (agents) => set({ agents }),

  addAgent: (agent) => set((state) => ({
    agents: [...state.agents, agent]
  })),

  updateAgent: (id, updates) => set((state) => ({
    agents: state.agents.map((a) =>
      a.id === id ? { ...a, ...updates } : a
    )
  })),

  removeAgent: (id) => set((state) => ({
    agents: state.agents.filter((a) => a.id !== id),
    selectedAgentId: state.selectedAgentId === id ? null : state.selectedAgentId
  })),

  selectAgent: (id) => set({ selectedAgentId: id }),

  addEvent: (event) => set((state) => ({
    events: [...state.events.slice(-99), event] // Keep last 100 events
  })),

  clearEvents: () => set({ events: [] })
}))

// API functions
const API_BASE = '/api'

export async function fetchAgents(): Promise<Agent[]> {
  const res = await fetch(`${API_BASE}/agents`)
  return res.json()
}

export async function spawnAgent(req: SpawnRequest): Promise<Agent> {
  const res = await fetch(`${API_BASE}/agents`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req)
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function killAgent(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/agents/${id}`, {
    method: 'DELETE'
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export async function sendMessage(id: string, content: string): Promise<void> {
  const res = await fetch(`${API_BASE}/agents/${id}/message`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content })
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}
