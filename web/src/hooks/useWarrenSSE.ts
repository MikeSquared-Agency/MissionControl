import { useEffect, useRef } from 'react'
import { useSwarmStore } from '../stores/useSwarmStore'
import type { WarrenSSEEvent } from '../types/swarm'

const MAX_RECONNECT_DELAY = 30_000
const INITIAL_RECONNECT_DELAY = 1_000

export function useWarrenSSE() {
  const esRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const attemptsRef = useRef(0)

  useEffect(() => {
    let mounted = true

    function connect() {
      if (!mounted) return
      if (esRef.current) {
        esRef.current.close()
      }

      const es = new EventSource('/api/swarm/warren/events')
      esRef.current = es

      es.onopen = () => {
        attemptsRef.current = 0
      }

      es.onmessage = (e) => {
        if (!mounted) return
        try {
          const parsed = JSON.parse(e.data)
          const event: WarrenSSEEvent = {
            id: parsed.id || e.lastEventId || undefined,
            type: parsed.type || 'unknown',
            agent: parsed.agent || parsed.agent_id,
            data: parsed.data ?? parsed,
            timestamp: parsed.timestamp || Date.now()
          }
          useSwarmStore.getState().addEvent(event)
        } catch {
          // Non-JSON SSE data — wrap as raw event
          useSwarmStore.getState().addEvent({
            type: 'raw',
            data: e.data,
            timestamp: Date.now()
          })
        }
      }

      es.onerror = () => {
        if (!mounted) return
        es.close()
        esRef.current = null

        // Exponential backoff
        const delay = Math.min(
          INITIAL_RECONNECT_DELAY * Math.pow(2, attemptsRef.current),
          MAX_RECONNECT_DELAY
        )
        attemptsRef.current++

        reconnectTimeoutRef.current = window.setTimeout(connect, delay)
      }
    }

    connect()

    return () => {
      mounted = false
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (esRef.current) {
        esRef.current.close()
        esRef.current = null
      }
    }
  }, [])
}
