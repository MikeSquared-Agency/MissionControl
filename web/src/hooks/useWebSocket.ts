import { useEffect, useRef, useCallback } from 'react'
import { useStore } from '../stores/useStore'
import { useWorkflowStore } from '../stores/useWorkflowStore'
import { useKnowledgeStore } from '../stores/useKnowledgeStore'
import type { Agent, Zone, ConversationMessage, ToolCall } from '../types'
import type { V4Event } from '../types/v4'

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const reconnectAttempts = useRef(0)
  const maxReconnectDelay = 30000

  const {
    setAgents,
    addAgent,
    updateAgent,
    removeAgent,
    setZones,
    addZone,
    updateZone,
    removeZone,
    addMessage,
    updateToolCall,
    setAgentAttention,
    addKingMessage,
    setConnectionStatus
  } = useStore()

  const connectionStatus = useStore((s) => s.connectionStatus)

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    setConnectionStatus('connecting')

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setConnectionStatus('connected')
      reconnectAttempts.current = 0
      console.log('WebSocket connected')
    }

    ws.onclose = () => {
      setConnectionStatus('disconnected')
      console.log('WebSocket disconnected')

      // Exponential backoff for reconnection
      const delay = Math.min(
        1000 * Math.pow(2, reconnectAttempts.current),
        maxReconnectDelay
      )
      reconnectAttempts.current++

      reconnectTimeoutRef.current = window.setTimeout(() => {
        connect()
      }, delay)
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        handleMessage(data)
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e)
      }
    }
  }, [setConnectionStatus, setAgents, addAgent, updateAgent, removeAgent, setZones, addZone, updateZone, removeZone, addMessage, updateToolCall, setAgentAttention, addKingMessage])

  const handleMessage = useCallback((data: WebSocketMessage) => {
    switch (data.type) {
      // Agent list (initial sync)
      case 'agent_list':
        if (data.agents) {
          setAgents(data.agents.map(normalizeAgent))
        }
        break

      // Agent spawned
      case 'agent_spawned':
        if (data.agent) {
          addAgent(normalizeAgent(data.agent))
        }
        break

      // Agent status changed
      case 'agent_status':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          updateAgent(agentId!, {
            status: normalizeStatus(data.status)
          })
        }
        break

      // Agent stopped
      case 'agent_stopped':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          const stoppedData = typeof data.data === 'object' ? data.data as Record<string, unknown> : {}
          updateAgent(agentId!, {
            status: normalizeStatus(stoppedData.status as string || data.status || 'stopped'),
            error: (stoppedData.error as string) || data.error
          })
        }
        break

      // Agent killed
      case 'agent_killed':
      case 'agent_removed':
        if (data.agent_id || data.agentId) {
          removeAgent(data.agent_id || data.agentId!)
        }
        break

      // Tool call
      case 'tool_call':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          const toolCall: ToolCall = {
            id: data.toolCallId || `tc-${Date.now()}`,
            tool: data.tool || 'unknown',
            args: data.args || {},
            collapsed: true
          }
          const message: ConversationMessage = {
            role: 'assistant',
            content: '',
            toolCalls: [toolCall],
            timestamp: Date.now()
          }
          addMessage(agentId!, message)
        }
        break

      // Tool result
      case 'tool_result':
        if ((data.agent_id || data.agentId) && data.toolCallId) {
          updateToolCall(data.agent_id || data.agentId!, data.toolCallId, data.result || '')
        }
        break

      // Message (conversation)
      case 'message':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          const msg: ConversationMessage = {
            role: data.role || 'assistant',
            content: data.content || '',
            isQuestion: data.isQuestion,
            isPermission: data.isPermission,
            timestamp: data.timestamp || Date.now()
          }
          addMessage(agentId!, msg)
        }
        break

      // Attention request
      case 'attention':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          setAgentAttention(agentId!, data.attention || null)
        }
        break

      // Tokens updated
      case 'tokens_updated':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          updateAgent(agentId!, {
            tokens: data.tokens || 0,
            cost: data.cost || 0
          })
        }
        break

      // Zone events
      case 'zone_created':
        if (data.zone) {
          addZone(data.zone)
        }
        break

      case 'zone_updated':
        if (data.zone) {
          updateZone(data.zone.id, data.zone)
        }
        break

      case 'zone_deleted':
        if (data.zoneId) {
          removeZone(data.zoneId)
        }
        break

      // Zone list (sync)
      case 'zone_list':
        if (data.zones) {
          setZones(data.zones)
        }
        break

      // King response
      case 'king_response':
        if (data.message) {
          addKingMessage(data.message)
        }
        break

      // Full sync
      case 'sync':
        if (data.agents) {
          setAgents(data.agents.map(normalizeAgent))
        }
        if (data.zones) {
          setZones(data.zones)
        }
        break

      // ============================================================
      // V4 Events
      // ============================================================

      // V4 initial state sync
      case 'v4_state':
        useWorkflowStore.getState().handleEvent(data as V4Event)
        break

      // V4 Workflow events
      case 'phase_changed':
        useWorkflowStore.getState().handleEvent(data as V4Event)
        break

      case 'task_created':
        useWorkflowStore.getState().handleEvent(data as V4Event)
        break

      case 'task_updated':
        useWorkflowStore.getState().handleEvent(data as V4Event)
        break

      case 'gate_status':
        useWorkflowStore.getState().handleEvent(data as V4Event)
        break

      // V4 Knowledge events
      case 'token_warning':
      case 'token_critical':
        useKnowledgeStore.getState().handleEvent(data as V4Event)
        break

      case 'checkpoint_created':
        useWorkflowStore.getState().handleEvent(data as V4Event)
        useKnowledgeStore.getState().handleEvent(data as V4Event)
        break

      case 'handoff_received':
        useKnowledgeStore.getState().handleEvent(data as V4Event)
        break

      case 'handoff_validated':
        // Could trigger UI notification
        console.log('Handoff validated:', data)
        break

      // V4 Runtime events
      case 'agent_health':
        // Update agent health status
        if (data.agent_id) {
          updateAgent(data.agent_id, {
            status: data.health === 'dead' ? 'stopped' :
                    data.health === 'stuck' ? 'error' :
                    data.health === 'idle' ? 'idle' : 'working'
          })
        }
        break

      case 'agent_stuck':
        // Mark agent as needing attention
        if (data.agent_id) {
          setAgentAttention(data.agent_id, {
            type: 'error',
            message: `Agent stuck for ${Math.floor((data.since_ms || 0) / 1000)}s`,
            since: Date.now() - (data.since_ms || 0)
          })
        }
        break

      // Legacy agent output (backward compatibility)
      case 'agent_output':
      case 'agent_error':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          const outputData = typeof data.data === 'object' ? data.data as Record<string, unknown> : { text: String(data.data) }

          // Check if it's a structured event
          if (outputData.type === 'tool_call') {
            const toolCall: ToolCall = {
              id: (outputData.id as string) || `tc-${Date.now()}`,
              tool: (outputData.tool as string) || 'unknown',
              args: (outputData.args as Record<string, unknown>) || {},
              collapsed: true
            }
            const message: ConversationMessage = {
              role: 'assistant',
              content: '',
              toolCalls: [toolCall],
              timestamp: Date.now()
            }
            addMessage(agentId!, message)
          } else if (outputData.type === 'output' || outputData.text) {
            const content = (outputData.content as string) || (outputData.text as string) || ''
            if (content) {
              const message: ConversationMessage = {
                role: data.type === 'agent_error' ? 'error' : 'assistant',
                content,
                timestamp: Date.now()
              }
              addMessage(agentId!, message)
            }
          }
        }
        break

      // Legacy turn event
      case 'turn':
        // Turn markers can be ignored for now
        break

      // Legacy thinking event
      case 'thinking':
        // Could be added to conversation if needed
        break

      // Legacy output event
      case 'output':
        if (data.agent_id || data.agentId) {
          const agentId = data.agent_id || data.agentId
          if (data.content) {
            const message: ConversationMessage = {
              role: 'assistant',
              content: data.content,
              timestamp: Date.now()
            }
            addMessage(agentId!, message)
          }
        }
        break

      default:
        console.log('Unknown WebSocket event type:', data.type, data)
    }
  }, [setAgents, addAgent, updateAgent, removeAgent, setZones, addZone, updateZone, removeZone, addMessage, updateToolCall, setAgentAttention, addKingMessage])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    wsRef.current?.close()
    wsRef.current = null
    setConnectionStatus('disconnected')
  }, [setConnectionStatus])

  const send = useCallback((data: object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  // Force reconnect
  const forceReconnect = useCallback(() => {
    disconnect()
    reconnectAttempts.current = 0
    connect()
  }, [disconnect, connect])

  useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  return { status: connectionStatus, send, connect, disconnect, forceReconnect }
}

// Type for WebSocket messages
interface WebSocketMessage {
  type: string
  agent_id?: string
  agentId?: string
  agents?: Record<string, unknown>[]
  agent?: Record<string, unknown>
  zones?: Zone[]
  zone?: Zone
  zoneId?: string
  status?: string
  error?: string
  tool?: string
  args?: Record<string, unknown>
  result?: string
  toolCallId?: string
  content?: string
  role?: 'user' | 'assistant' | 'error'
  isQuestion?: boolean
  isPermission?: boolean
  timestamp?: number
  tokens?: number
  cost?: number
  attention?: Agent['attention']
  message?: import('../types').KingMessage
  data?: unknown

  // V4 fields
  state?: {
    current_phase: string
    phases: Array<{ phase: string; status: string }>
    tasks: unknown[]
    checkpoints: unknown[]
  }
  phase?: string
  previous?: string
  task?: unknown
  task_id?: string
  worker_id?: string
  criteria?: Array<{ description: string; satisfied: boolean }>
  usage?: number
  budget?: number
  remaining?: number
  checkpoint_id?: string
  valid?: boolean
  errors?: string[]
  health?: string
  since_ms?: number
}

// Helper to normalize agent from backend
function normalizeAgent(backendAgent: Record<string, unknown>): Agent {
  return {
    id: backendAgent.id as string,
    name: (backendAgent.name as string) || (backendAgent.id as string),
    type: normalizeType(backendAgent.type as string),
    persona: (backendAgent.persona as string) || null,
    status: normalizeStatus(backendAgent.status as string),
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

function normalizeType(type: string): Agent['type'] {
  if (type === 'claude' || type === 'claude-code') return 'claude-code'
  return 'python'
}

function normalizeStatus(status: string | undefined): Agent['status'] {
  if (!status) return 'idle'
  const statusMap: Record<string, Agent['status']> = {
    'starting': 'starting',
    'running': 'working',
    'working': 'working',
    'idle': 'idle',
    'error': 'error',
    'waiting': 'waiting',
    'stopped': 'stopped'
  }
  return statusMap[status] || 'idle'
}
