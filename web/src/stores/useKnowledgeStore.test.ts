import { describe, it, expect, beforeEach } from 'vitest'
import { useKnowledgeStore } from './useKnowledgeStore'
import type { TokenBudget, Finding, Handoff, SessionStatus, SessionRecord, WorkflowEvent } from '../types/workflow'

describe('useKnowledgeStore', () => {
  beforeEach(() => {
    // Reset store to initial state before each test
    useKnowledgeStore.setState({
      budgets: {},
      checkpoints: [],
      findings: [],
      recentHandoffs: [],
      sessionStatus: null,
      sessionHistory: [],
      loading: false,
      error: null
    })
  })

  describe('Budget actions', () => {
    const testBudget: TokenBudget = {
      worker_id: 'worker-1',
      budget: 20000,
      used: 5000,
      status: 'healthy',
      remaining: 15000
    }

    it('should set a budget', () => {
      useKnowledgeStore.getState().setBudget('worker-1', testBudget)

      expect(useKnowledgeStore.getState().budgets['worker-1']).toBeDefined()
      expect(useKnowledgeStore.getState().budgets['worker-1'].budget).toBe(20000)
    })

    it('should remove a budget', () => {
      useKnowledgeStore.getState().setBudget('worker-1', testBudget)
      useKnowledgeStore.getState().removeBudget('worker-1')

      expect(useKnowledgeStore.getState().budgets['worker-1']).toBeUndefined()
    })

    it('should track multiple budgets', () => {
      useKnowledgeStore.getState().setBudget('worker-1', testBudget)
      useKnowledgeStore.getState().setBudget('worker-2', { ...testBudget, worker_id: 'worker-2' })

      expect(Object.keys(useKnowledgeStore.getState().budgets)).toHaveLength(2)
    })
  })

  describe('Checkpoint actions', () => {
    it('should set checkpoints', () => {
      const checkpoints = [
        { id: 'cp-1', stage: 'discovery' as const, created_at: Date.now() }
      ]
      useKnowledgeStore.getState().setCheckpoints(checkpoints)

      expect(useKnowledgeStore.getState().checkpoints).toHaveLength(1)
    })

    it('should add a checkpoint', () => {
      useKnowledgeStore.getState().addCheckpoint({
        id: 'cp-2',
        stage: 'design',
        created_at: Date.now()
      })

      expect(useKnowledgeStore.getState().checkpoints).toHaveLength(1)
    })
  })

  describe('Finding actions', () => {
    it('should add a finding', () => {
      const finding: Finding = {
        type: 'discovery',
        summary: 'Found something important'
      }
      useKnowledgeStore.getState().addFinding(finding)

      expect(useKnowledgeStore.getState().findings).toHaveLength(1)
      expect(useKnowledgeStore.getState().findings[0].type).toBe('discovery')
    })
  })

  describe('Handoff actions', () => {
    it('should add a handoff', () => {
      const handoff: Handoff = {
        task_id: 'task-1',
        worker_id: 'worker-1',
        status: 'complete',
        findings: [],
        artifacts: [],
        timestamp: Date.now()
      }
      useKnowledgeStore.getState().addHandoff(handoff)

      expect(useKnowledgeStore.getState().recentHandoffs).toHaveLength(1)
    })

    it('should keep only last 50 handoffs', () => {
      for (let i = 0; i < 60; i++) {
        useKnowledgeStore.getState().addHandoff({
          task_id: `task-${i}`,
          worker_id: 'worker-1',
          status: 'complete',
          findings: [],
          artifacts: [],
          timestamp: Date.now()
        })
      }

      expect(useKnowledgeStore.getState().recentHandoffs).toHaveLength(50)
    })

    it('should add newest handoffs first', () => {
      useKnowledgeStore.getState().addHandoff({
        task_id: 'task-old',
        worker_id: 'worker-1',
        status: 'complete',
        findings: [],
        artifacts: [],
        timestamp: Date.now()
      })
      useKnowledgeStore.getState().addHandoff({
        task_id: 'task-new',
        worker_id: 'worker-1',
        status: 'complete',
        findings: [],
        artifacts: [],
        timestamp: Date.now()
      })

      expect(useKnowledgeStore.getState().recentHandoffs[0].task_id).toBe('task-new')
    })
  })

  describe('Event handling', () => {
    it('should handle token_warning event', () => {
      const event: WorkflowEvent = {
        type: 'token_warning',
        worker_id: 'worker-1',
        usage: 12000,
        budget: 20000,
        status: 'warning',
        remaining: 8000
      }

      useKnowledgeStore.getState().handleEvent(event)

      const budget = useKnowledgeStore.getState().budgets['worker-1']
      expect(budget).toBeDefined()
      expect(budget.status).toBe('warning')
      expect(budget.used).toBe(12000)
    })

    it('should handle token_critical event', () => {
      const event: WorkflowEvent = {
        type: 'token_critical',
        worker_id: 'worker-2',
        usage: 18000,
        budget: 20000,
        status: 'critical',
        remaining: 2000
      }

      useKnowledgeStore.getState().handleEvent(event)

      const budget = useKnowledgeStore.getState().budgets['worker-2']
      expect(budget).toBeDefined()
      expect(budget.status).toBe('critical')
    })

    it('should handle checkpoint_created event', () => {
      const event: WorkflowEvent = {
        type: 'checkpoint_created',
        checkpoint_id: 'cp-new',
        stage: 'design'
      }

      useKnowledgeStore.getState().handleEvent(event)

      expect(useKnowledgeStore.getState().checkpoints).toHaveLength(1)
    })

    it('should handle handoff_received event', () => {
      const event: WorkflowEvent = {
        type: 'handoff_received',
        task_id: 'task-1',
        worker_id: 'worker-1',
        status: 'complete'
      }

      useKnowledgeStore.getState().handleEvent(event)

      expect(useKnowledgeStore.getState().recentHandoffs).toHaveLength(1)
    })
  })

  describe('Session status', () => {
    const testStatus: SessionStatus = {
      session_id: 'sess-abc123',
      stage: 'implement',
      session_start: Date.now() - 3600000,
      duration_minutes: 60,
      last_checkpoint: 'cp-001',
      tasks_total: 10,
      tasks_complete: 4,
      health: 'green',
      recommendation: 'Session is healthy'
    }

    it('should set session status', () => {
      useKnowledgeStore.getState().setSessionStatus(testStatus)

      const status = useKnowledgeStore.getState().sessionStatus
      expect(status).toBeDefined()
      expect(status!.session_id).toBe('sess-abc123')
      expect(status!.health).toBe('green')
      expect(status!.stage).toBe('implement')
    })

    it('should update session status', () => {
      useKnowledgeStore.getState().setSessionStatus(testStatus)
      useKnowledgeStore.getState().setSessionStatus({
        ...testStatus,
        health: 'yellow',
        duration_minutes: 120,
        recommendation: 'Consider creating a checkpoint'
      })

      const status = useKnowledgeStore.getState().sessionStatus
      expect(status!.health).toBe('yellow')
      expect(status!.duration_minutes).toBe(120)
    })

    it('should start with null session status', () => {
      expect(useKnowledgeStore.getState().sessionStatus).toBeNull()
    })
  })

  describe('Session history', () => {
    const testSessions: SessionRecord[] = [
      {
        session_id: 'sess-001',
        started_at: Date.now() - 7200000,
        ended_at: Date.now() - 3600000,
        checkpoint_id: 'cp-001',
        stage: 'design',
        reason: 'manual'
      },
      {
        session_id: 'sess-002',
        started_at: Date.now() - 3600000,
        stage: 'implement'
      }
    ]

    it('should set session history', () => {
      useKnowledgeStore.getState().setSessionHistory(testSessions)

      const history = useKnowledgeStore.getState().sessionHistory
      expect(history).toHaveLength(2)
      expect(history[0].session_id).toBe('sess-001')
      expect(history[1].session_id).toBe('sess-002')
    })

    it('should identify current session (no ended_at)', () => {
      useKnowledgeStore.getState().setSessionHistory(testSessions)

      const history = useKnowledgeStore.getState().sessionHistory
      const current = history.find(s => !s.ended_at)
      expect(current).toBeDefined()
      expect(current!.session_id).toBe('sess-002')
    })

    it('should start with empty session history', () => {
      expect(useKnowledgeStore.getState().sessionHistory).toHaveLength(0)
    })
  })

  describe('Loading and error state', () => {
    it('should set loading state', () => {
      useKnowledgeStore.getState().setLoading(true)
      expect(useKnowledgeStore.getState().loading).toBe(true)
    })

    it('should set error state', () => {
      useKnowledgeStore.getState().setError('Test error')
      expect(useKnowledgeStore.getState().error).toBe('Test error')
    })
  })
})
