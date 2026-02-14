import { useEffect, useRef } from 'react'
import { useSwarmStore, fetchSwarmOverview } from '../stores/useSwarmStore'

const POLL_INTERVAL = 10_000 // 10 seconds

interface UseSwarmPollingOptions {
  enabled: boolean
}

export function useSwarmPolling({ enabled }: UseSwarmPollingOptions) {
  const intervalRef = useRef<number | null>(null)
  const mountedRef = useRef(true)

  useEffect(() => {
    mountedRef.current = true

    if (!enabled) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      return
    }

    const poll = async () => {
      if (!mountedRef.current) return

      const store = useSwarmStore.getState()
      // Only set loading on first fetch
      if (!store.lastFetched) {
        store.setLoading(true)
      }

      try {
        const data = await fetchSwarmOverview()
        if (mountedRef.current) {
          store.setOverview(data)
        }
      } catch (err) {
        if (mountedRef.current) {
          store.setError(err instanceof Error ? err.message : 'Failed to fetch swarm overview')
        }
      }
    }

    // Fetch immediately
    poll()

    // Then poll on interval
    intervalRef.current = window.setInterval(poll, POLL_INTERVAL)

    return () => {
      mountedRef.current = false
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [enabled])
}
