import { useEffect, useRef, useState, useCallback } from 'react'
import { useAgentStore } from '../stores/agentStore'
import type { Event, Agent } from '../types'

type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const reconnectTimeoutRef = useRef<number | null>(null)

  const { setAgents, addAgent, updateAgent, removeAgent, addEvent } = useAgentStore()

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    setStatus('connecting')

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('connected')
      console.log('WebSocket connected')
    }

    ws.onclose = () => {
      setStatus('disconnected')
      console.log('WebSocket disconnected')

      // Reconnect after 3 seconds
      reconnectTimeoutRef.current = window.setTimeout(() => {
        connect()
      }, 3000)
    }

    ws.onerror = (error) => {
      console.error('WebSocket error:', error)
      setStatus('error')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        handleMessage(data)
      } catch (e) {
        console.error('Failed to parse WebSocket message:', e)
      }
    }
  }, [setAgents, addAgent, updateAgent, removeAgent, addEvent])

  const handleMessage = (data: Event & { type: string; agents?: Agent[]; agent?: Agent }) => {
    switch (data.type) {
      case 'agent_list':
        if (data.agents) {
          setAgents(data.agents)
        }
        break

      case 'agent_spawned':
        if (data.agent) {
          addAgent(data.agent)
        }
        break

      case 'agent_stopped':
        if (data.agent_id) {
          updateAgent(data.agent_id, {
            status: data.status as Agent['status'],
            error: data.error
          })
        }
        break

      case 'agent_output':
      case 'agent_error':
      case 'tool_call':
      case 'tool_result':
      case 'turn':
      case 'thinking':
      case 'output':
        addEvent(data)
        break

      default:
        console.log('Unknown event type:', data.type, data)
    }
  }

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    wsRef.current?.close()
    wsRef.current = null
    setStatus('disconnected')
  }, [])

  const send = useCallback((data: object) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    }
  }, [])

  useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  return { status, send, connect, disconnect }
}
