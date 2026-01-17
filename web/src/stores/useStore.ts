import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type {
  Agent,
  Zone,
  Persona,
  KingMessage,
  ConversationMessage,
  ConnectionStatus,
  Settings,
  SpawnRequest
} from '../types'
import { DEFAULT_PERSONAS, DEFAULT_ZONE } from '../types'

// Re-export defaults
export { DEFAULT_PERSONAS, DEFAULT_ZONE }

interface AppState {
  // Connection
  connectionStatus: ConnectionStatus

  // Core data
  agents: Agent[]
  zones: Zone[]
  personas: Persona[]

  // Selection
  selectedAgentId: string | null
  collapsedZones: Record<string, boolean>

  // King mode
  kingMode: boolean
  kingConversation: KingMessage[]

  // Settings
  settings: Settings

  // Agent actions
  setAgents: (agents: Agent[]) => void
  addAgent: (agent: Agent) => void
  updateAgent: (id: string, updates: Partial<Agent>) => void
  removeAgent: (id: string) => void

  // Zone actions
  setZones: (zones: Zone[]) => void
  addZone: (zone: Zone) => void
  updateZone: (id: string, updates: Partial<Zone>) => void
  removeZone: (id: string) => void

  // Persona actions
  setPersonas: (personas: Persona[]) => void
  addPersona: (persona: Persona) => void
  updatePersona: (id: string, updates: Partial<Persona>) => void
  removePersona: (id: string) => void

  // Selection actions
  selectAgent: (id: string | null) => void
  toggleZoneCollapse: (id: string) => void

  // King mode actions
  setKingMode: (enabled: boolean) => void
  addKingMessage: (message: KingMessage) => void
  clearKingConversation: () => void

  // Settings actions
  updateSettings: (updates: Partial<Settings>) => void

  // Connection action
  setConnectionStatus: (status: ConnectionStatus) => void

  // Conversation actions
  addMessage: (agentId: string, message: ConversationMessage) => void
  updateToolCall: (agentId: string, toolCallId: string, result: string) => void
  toggleToolCallCollapse: (agentId: string, toolCallId: string) => void
  setAgentAttention: (agentId: string, attention: Agent['attention']) => void
  addFinding: (agentId: string, finding: string) => void
  clearConversation: (agentId: string) => void
}

// Import defaults at runtime
import { DEFAULT_PERSONAS as defaultPersonas, DEFAULT_ZONE as defaultZone } from '../types'

export const useStore = create<AppState>()(
  persist(
    (set, get) => ({
      // Initial state
      connectionStatus: 'disconnected',
      agents: [],
      zones: [defaultZone],
      personas: [...defaultPersonas],
      selectedAgentId: null,
      collapsedZones: {},
      kingMode: false,
      kingConversation: [],
      settings: {
        apiKey: '',
        defaultWorkingDir: ''
      },

      // Agent actions
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

      // Zone actions
      setZones: (zones) => set({ zones }),

      addZone: (zone) => set((state) => ({
        zones: [...state.zones, zone]
      })),

      updateZone: (id, updates) => set((state) => ({
        zones: state.zones.map((z) =>
          z.id === id ? { ...z, ...updates } : z
        )
      })),

      removeZone: (id) => set((state) => ({
        zones: state.zones.filter((z) => z.id !== id),
        collapsedZones: Object.fromEntries(
          Object.entries(state.collapsedZones).filter(([key]) => key !== id)
        )
      })),

      // Persona actions
      setPersonas: (personas) => set({ personas }),

      addPersona: (persona) => set((state) => ({
        personas: [...state.personas, persona]
      })),

      updatePersona: (id, updates) => set((state) => ({
        personas: state.personas.map((p) =>
          p.id === id ? { ...p, ...updates } : p
        )
      })),

      removePersona: (id) => set((state) => ({
        personas: state.personas.filter((p) => p.id !== id)
      })),

      // Selection actions
      selectAgent: (id) => set({ selectedAgentId: id }),

      toggleZoneCollapse: (id) => set((state) => ({
        collapsedZones: {
          ...state.collapsedZones,
          [id]: !state.collapsedZones[id]
        }
      })),

      // King mode actions
      setKingMode: (enabled) => set({
        kingMode: enabled,
        selectedAgentId: enabled ? null : get().selectedAgentId
      }),

      addKingMessage: (message) => set((state) => ({
        kingConversation: [...state.kingConversation, message]
      })),

      clearKingConversation: () => set({ kingConversation: [] }),

      // Settings actions
      updateSettings: (updates) => set((state) => ({
        settings: { ...state.settings, ...updates }
      })),

      // Connection action
      setConnectionStatus: (status) => set({ connectionStatus: status }),

      // Conversation actions
      addMessage: (agentId, message) => set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId
            ? { ...a, conversation: [...a.conversation, message] }
            : a
        )
      })),

      updateToolCall: (agentId, toolCallId, result) => set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId
            ? {
                ...a,
                conversation: a.conversation.map((msg) =>
                  msg.toolCalls
                    ? {
                        ...msg,
                        toolCalls: msg.toolCalls.map((tc) =>
                          tc.id === toolCallId ? { ...tc, result } : tc
                        )
                      }
                    : msg
                )
              }
            : a
        )
      })),

      toggleToolCallCollapse: (agentId, toolCallId) => set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId
            ? {
                ...a,
                conversation: a.conversation.map((msg) =>
                  msg.toolCalls
                    ? {
                        ...msg,
                        toolCalls: msg.toolCalls.map((tc) =>
                          tc.id === toolCallId ? { ...tc, collapsed: !tc.collapsed } : tc
                        )
                      }
                    : msg
                )
              }
            : a
        )
      })),

      setAgentAttention: (agentId, attention) => set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId
            ? {
                ...a,
                attention,
                status: attention ? 'waiting' : a.status === 'waiting' ? 'working' : a.status
              }
            : a
        )
      })),

      addFinding: (agentId, finding) => set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId
            ? { ...a, findings: [...a.findings, finding] }
            : a
        )
      })),

      clearConversation: (agentId) => set((state) => ({
        agents: state.agents.map((a) =>
          a.id === agentId
            ? { ...a, conversation: [], findings: [] }
            : a
        )
      }))
    }),
    {
      name: 'mission-control-storage',
      partialize: (state) => ({
        settings: state.settings,
        personas: state.personas,
        collapsedZones: state.collapsedZones,
        kingMode: state.kingMode,
        zones: state.zones
      })
    }
  )
)

// API functions
const API_BASE = '/api'

export async function fetchAgents(): Promise<Agent[]> {
  const res = await fetch(`${API_BASE}/agents`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
  const agents = await res.json()
  // Normalize agent data from backend
  return agents.map(normalizeAgent)
}

export async function fetchZones(): Promise<Zone[]> {
  const res = await fetch(`${API_BASE}/zones`)
  if (!res.ok) {
    throw new Error(await res.text())
  }
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
  return normalizeAgent(await res.json())
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

export async function createZone(zone: Omit<Zone, 'id'>): Promise<Zone> {
  const res = await fetch(`${API_BASE}/zones`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(zone)
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function updateZoneApi(id: string, updates: Partial<Zone>): Promise<Zone> {
  const res = await fetch(`${API_BASE}/zones/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates)
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
  return res.json()
}

export async function deleteZone(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/zones/${id}`, {
    method: 'DELETE'
  })
  if (!res.ok) {
    throw new Error(await res.text())
  }
}

export async function sendKingMessage(content: string): Promise<void> {
  const res = await fetch(`${API_BASE}/king/message`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content })
  })

  if (!res.ok) {
    throw new Error(await res.text())
  }

  // Response will come through WebSocket as king_response event
}

export async function respondToAttention(agentId: string, response: string): Promise<void> {
  // Clear attention locally first for immediate feedback
  useStore.getState().setAgentAttention(agentId, null)

  // If it's just a dismiss, we don't need to send to backend
  if (response === 'dismiss') {
    return
  }

  // Send the response to the agent
  const res = await fetch(`${API_BASE}/agents/${agentId}/respond`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ response })
  })

  if (!res.ok) {
    // Restore attention if request failed - we'd need to refetch
    throw new Error(await res.text())
  }
}

// Normalize agent data from backend to match our frontend types
function normalizeAgent(backendAgent: Record<string, unknown>): Agent {
  return {
    id: backendAgent.id as string,
    name: (backendAgent.name as string) || (backendAgent.id as string),
    type: normalizeAgentType(backendAgent.type as string),
    persona: (backendAgent.persona as string) || null,
    status: normalizeAgentStatus(backendAgent.status as string),
    tokens: (backendAgent.tokens as number) || 0,
    cost: (backendAgent.cost as number) || 0,
    task: (backendAgent.task as string) || '',
    zone: (backendAgent.zone as string) || 'default',
    workingDir: (backendAgent.workingDir as string) || (backendAgent.workdir as string) || '',
    attention: null,
    findings: [],
    conversation: [],
    created_at: (backendAgent.created_at as string) || new Date().toISOString(),
    error: backendAgent.error as string | undefined,
    pid: backendAgent.pid as number | undefined
  }
}

function normalizeAgentType(type: string): Agent['type'] {
  if (type === 'claude' || type === 'claude-code') return 'claude-code'
  return 'python'
}

function normalizeAgentStatus(status: string): Agent['status'] {
  const validStatuses: Agent['status'][] = ['starting', 'working', 'idle', 'error', 'waiting', 'stopped']
  if (validStatuses.includes(status as Agent['status'])) {
    return status as Agent['status']
  }
  // Map 'running' to 'working'
  if (status === 'running') return 'working'
  return 'idle'
}

// Selectors
export const useAgents = () => useStore((s) => s.agents)
export const useZones = () => useStore((s) => s.zones)
export const usePersonas = () => useStore((s) => s.personas)
export const useSelectedAgent = () => {
  const agents = useStore((s) => s.agents)
  const selectedId = useStore((s) => s.selectedAgentId)
  return agents.find((a) => a.id === selectedId) || null
}
export const useConnectionStatus = () => useStore((s) => s.connectionStatus)
export const useKingMode = () => useStore((s) => s.kingMode)
export const useSettings = () => useStore((s) => s.settings)

// Computed selectors - these just return agents, components should filter with useMemo
export const useAgentsByZone = (zoneId: string) => {
  const agents = useStore((s) => s.agents)
  // Filter outside selector to avoid new array reference issues
  // Components using this should memoize if needed
  return agents.filter((a) => a.zone === zoneId)
}

export const useAgentsNeedingAttention = () => {
  const agents = useStore((s) => s.agents)
  return agents.filter((a) => a.attention !== null)
}

// Stats selector - compute outside to avoid infinite loop
export const useStats = () => {
  const agents = useStore((s) => s.agents)

  // Compute stats from agents array
  const total = agents.length
  const working = agents.filter((a) => a.status === 'working').length
  const waiting = agents.filter((a) => a.status === 'waiting' || a.attention !== null).length
  const error = agents.filter((a) => a.status === 'error').length
  const tokens = agents.reduce((sum, a) => sum + a.tokens, 0)
  const cost = agents.reduce((sum, a) => sum + a.cost, 0)

  return { total, working, waiting, error, tokens, cost }
}
