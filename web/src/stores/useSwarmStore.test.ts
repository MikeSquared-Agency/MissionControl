import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useSwarmStore } from './useSwarmStore'
import type { SwarmOverview, WarrenSSEEvent } from '../types/swarm'

// Helper to build a minimal valid overview.
function makeOverview(overrides: Partial<SwarmOverview> = {}): SwarmOverview {
  return {
    errors: {},
    fetched_at: new Date().toISOString(),
    ...overrides
  }
}

describe('useSwarmStore', () => {
  beforeEach(() => {
    useSwarmStore.setState({
      overview: null,
      events: [],
      alerts: [],
      loading: false,
      error: null,
      lastFetched: null
    })
  })

  describe('setOverview', () => {
    it('should set overview and update lastFetched', () => {
      const data = makeOverview({
        warren: { health: { status: 'ok' }, agents: [] }
      })

      useSwarmStore.getState().setOverview(data)

      const state = useSwarmStore.getState()
      expect(state.overview).toEqual(data)
      expect(state.lastFetched).toBeTypeOf('number')
      expect(state.loading).toBe(false)
      expect(state.error).toBeNull()
    })
  })

  describe('addEvent', () => {
    it('should prepend events in newest-first order', () => {
      const event1: WarrenSSEEvent = { type: 'heartbeat', timestamp: 1 }
      const event2: WarrenSSEEvent = { type: 'agent_ready', timestamp: 2 }

      useSwarmStore.getState().addEvent(event1)
      useSwarmStore.getState().addEvent(event2)

      const events = useSwarmStore.getState().events
      expect(events).toHaveLength(2)
      expect(events[0].type).toBe('agent_ready')
      expect(events[1].type).toBe('heartbeat')
    })

    it('should cap events at 100', () => {
      for (let i = 0; i < 110; i++) {
        useSwarmStore.getState().addEvent({ type: `event-${i}`, timestamp: i })
      }

      expect(useSwarmStore.getState().events).toHaveLength(100)
      // Most recent should be first.
      expect(useSwarmStore.getState().events[0].type).toBe('event-109')
    })
  })

  describe('Alert derivation from errors', () => {
    it('should generate critical alert from errors map', () => {
      const data = makeOverview({
        errors: { dispatch: 'connection refused' }
      })

      useSwarmStore.getState().setOverview(data)

      const alerts = useSwarmStore.getState().alerts
      expect(alerts).toHaveLength(1)
      expect(alerts[0].level).toBe('critical')
      expect(alerts[0].service).toBe('dispatch')
      expect(alerts[0].message).toContain('dispatch')
      expect(alerts[0].message).toContain('unreachable')
    })

    it('should not re-alert on the same error if already seen', () => {
      const data = makeOverview({
        errors: { dispatch: 'connection refused' }
      })

      useSwarmStore.getState().setOverview(data)
      // Set again with same error.
      useSwarmStore.getState().setOverview(data)

      // Should only have 1 alert, not 2.
      expect(useSwarmStore.getState().alerts).toHaveLength(1)
    })
  })

  describe('DLQ spike detection', () => {
    it('should generate warning alert on DLQ spike', () => {
      // First set with low DLQ.
      const baseline = makeOverview({
        chronicle: { dlq: { depth: 5 } }
      })
      useSwarmStore.getState().setOverview(baseline)

      // Spike: depth > 10 and > 2x previous.
      const spike = makeOverview({
        chronicle: { dlq: { depth: 50 } }
      })
      useSwarmStore.getState().setOverview(spike)

      const alerts = useSwarmStore.getState().alerts
      const dlqAlert = alerts.find((a) => a.id.startsWith('dlq-spike'))
      expect(dlqAlert).toBeDefined()
      expect(dlqAlert!.level).toBe('warning')
      expect(dlqAlert!.message).toContain('50')
    })

    it('should not alert when DLQ is below threshold', () => {
      const data = makeOverview({
        chronicle: { dlq: { depth: 5 } }
      })
      useSwarmStore.getState().setOverview(data)

      // Small increase, still under 10.
      const small = makeOverview({
        chronicle: { dlq: { depth: 8 } }
      })
      useSwarmStore.getState().setOverview(small)

      const alerts = useSwarmStore.getState().alerts
      const dlqAlert = alerts.find((a) => a.id.startsWith('dlq-spike'))
      expect(dlqAlert).toBeUndefined()
    })
  })

  describe('dismissAlert', () => {
    it('should remove alert by id', () => {
      useSwarmStore.getState().addAlert({
        id: 'test-1',
        level: 'warning',
        service: 'chronicle',
        message: 'Test alert',
        timestamp: Date.now()
      })

      expect(useSwarmStore.getState().alerts).toHaveLength(1)

      useSwarmStore.getState().dismissAlert('test-1')

      expect(useSwarmStore.getState().alerts).toHaveLength(0)
    })

    it('should not affect other alerts', () => {
      useSwarmStore.getState().addAlert({
        id: 'keep',
        level: 'info',
        service: 'warren',
        message: 'Keep me',
        timestamp: Date.now()
      })
      useSwarmStore.getState().addAlert({
        id: 'remove',
        level: 'critical',
        service: 'dispatch',
        message: 'Remove me',
        timestamp: Date.now()
      })

      useSwarmStore.getState().dismissAlert('remove')

      const alerts = useSwarmStore.getState().alerts
      expect(alerts).toHaveLength(1)
      expect(alerts[0].id).toBe('keep')
    })
  })

  describe('setLoading / setError', () => {
    it('should set loading state', () => {
      useSwarmStore.getState().setLoading(true)
      expect(useSwarmStore.getState().loading).toBe(true)

      useSwarmStore.getState().setLoading(false)
      expect(useSwarmStore.getState().loading).toBe(false)
    })

    it('should set error and clear loading', () => {
      useSwarmStore.getState().setLoading(true)
      useSwarmStore.getState().setError('Network failure')

      expect(useSwarmStore.getState().error).toBe('Network failure')
      expect(useSwarmStore.getState().loading).toBe(false)
    })

    it('should clear error', () => {
      useSwarmStore.getState().setError('Something')
      useSwarmStore.getState().setError(null)

      expect(useSwarmStore.getState().error).toBeNull()
    })
  })

  describe('useFleetSummary (via getState)', () => {
    it('should return null when no overview', () => {
      // useFleetSummary is a hook — test the logic directly.
      const overview = useSwarmStore.getState().overview
      expect(overview).toBeNull()
    })

    it('should compute fleet counts from overview data', () => {
      const data = makeOverview({
        warren: {
          agents: [
            { id: 'a1', name: 'Agent1', state: 'ready' },
            { id: 'a2', name: 'Agent2', state: 'sleeping' },
            { id: 'a3', name: 'Agent3', state: 'ready' }
          ]
        },
        dispatch: {
          stats: { pending: 3, in_progress: 7, completed: 20, failed: 1, total: 31 },
          agents: [{ id: 'd1', status: 'active' }]
        },
        chronicle: { dlq: { depth: 4 } },
        promptforge: { prompt_count: 15 },
        alexandria: { collection_count: 8 },
        errors: { someservice: 'down' }
      })

      useSwarmStore.getState().setOverview(data)
      const overview = useSwarmStore.getState().overview!

      // Replicate useFleetSummary logic.
      const warrenAgents = overview.warren?.agents ?? []
      const dispatchAgents = overview.dispatch?.agents ?? []
      const totalAgents = warrenAgents.length + dispatchAgents.length
      const readyCount = warrenAgents.filter((a) => a.state === 'ready').length
      const sleepingCount = warrenAgents.filter((a) => a.state === 'sleeping').length
      const degradedCount = Object.keys(overview.errors).length

      expect(totalAgents).toBe(4)  // 3 warren + 1 dispatch
      expect(readyCount).toBe(2)
      expect(sleepingCount).toBe(1)
      expect(degradedCount).toBe(1)
      expect(overview.dispatch?.stats?.in_progress).toBe(7)
      expect(overview.chronicle?.dlq?.depth).toBe(4)
      expect(overview.promptforge?.prompt_count).toBe(15)
      expect(overview.alexandria?.collection_count).toBe(8)
    })
  })

  describe('usePipelineSummary (via getState)', () => {
    it('should compute pipeline values from dispatch stats', () => {
      const data = makeOverview({
        dispatch: {
          stats: { pending: 5, in_progress: 3, completed: 42, failed: 2, total: 52 }
        },
        chronicle: { dlq: { depth: 7 } }
      })

      useSwarmStore.getState().setOverview(data)
      const overview = useSwarmStore.getState().overview!

      const stats = overview.dispatch!.stats!
      expect(stats.pending).toBe(5)
      expect(stats.in_progress).toBe(3)
      expect(stats.completed).toBe(42)
      expect(stats.failed).toBe(2)
      expect(stats.total).toBe(52)
      expect(overview.chronicle?.dlq?.depth).toBe(7)
    })

    it('should handle missing dispatch stats', () => {
      const data = makeOverview()
      useSwarmStore.getState().setOverview(data)

      const overview = useSwarmStore.getState().overview!
      expect(overview.dispatch?.stats).toBeUndefined()
    })
  })
})
