import { describe, it, expect, beforeEach } from 'vitest'
import { useWorkflowStore } from './useWorkflowStore'
import type { Task, Phase, WorkflowEvent } from '../types/workflow'

describe('useWorkflowStore', () => {
  beforeEach(() => {
    // Reset store to initial state before each test
    useWorkflowStore.setState({
      currentPhase: 'idea',
      phases: [],
      tasks: [],
      gates: {},
      checkpoints: [],
      loading: false,
      error: null
    })
  })

  describe('Phase actions', () => {
    it('should set phases', () => {
      const phases = [
        { phase: 'idea' as Phase, status: 'current' as const },
        { phase: 'design' as Phase, status: 'pending' as const }
      ]
      useWorkflowStore.getState().setPhases('idea', phases)

      expect(useWorkflowStore.getState().currentPhase).toBe('idea')
      expect(useWorkflowStore.getState().phases).toHaveLength(2)
    })
  })

  describe('Task actions', () => {
    const testTask: Task = {
      id: 'task-1',
      name: 'Test Task',
      phase: 'idea',
      zone: 'frontend',
      status: 'pending',
      persona: 'developer',
      dependencies: [],
      created_at: Date.now(),
      updated_at: Date.now()
    }

    it('should set tasks', () => {
      useWorkflowStore.getState().setTasks([testTask])
      expect(useWorkflowStore.getState().tasks).toHaveLength(1)
    })

    it('should add a task', () => {
      useWorkflowStore.getState().addTask(testTask)
      expect(useWorkflowStore.getState().tasks).toHaveLength(1)
      expect(useWorkflowStore.getState().tasks[0].name).toBe('Test Task')
    })

    it('should update a task', () => {
      useWorkflowStore.getState().addTask(testTask)
      useWorkflowStore.getState().updateTask('task-1', { status: 'in_progress' })

      expect(useWorkflowStore.getState().tasks[0].status).toBe('in_progress')
    })
  })

  describe('Gate actions', () => {
    it('should set a gate', () => {
      const gate = {
        id: 'gate-idea',
        phase: 'idea' as Phase,
        status: 'closed' as const,
        criteria: [{ description: 'Test criterion', satisfied: false }]
      }
      useWorkflowStore.getState().setGate('idea', gate)

      expect(useWorkflowStore.getState().gates['idea']).toBeDefined()
      expect(useWorkflowStore.getState().gates['idea'].status).toBe('closed')
    })
  })

  describe('Checkpoint actions', () => {
    it('should set checkpoints', () => {
      const checkpoints = [
        { id: 'cp-1', phase: 'idea' as Phase, created_at: Date.now() }
      ]
      useWorkflowStore.getState().setCheckpoints(checkpoints)

      expect(useWorkflowStore.getState().checkpoints).toHaveLength(1)
    })

    it('should add a checkpoint', () => {
      useWorkflowStore.getState().addCheckpoint({
        id: 'cp-2',
        phase: 'design',
        created_at: Date.now()
      })

      expect(useWorkflowStore.getState().checkpoints).toHaveLength(1)
    })
  })

  describe('Event handling', () => {
    it('should handle v4_state event', () => {
      const event: WorkflowEvent = {
        type: 'v4_state',
        state: {
          current_phase: 'design',
          phases: [
            { phase: 'idea', status: 'complete' },
            { phase: 'design', status: 'current' }
          ],
          tasks: [{
            id: 'task-1',
            name: 'Test',
            phase: 'design',
            zone: 'test',
            status: 'pending',
            persona: 'dev',
            dependencies: [],
            created_at: Date.now(),
            updated_at: Date.now()
          }],
          checkpoints: []
        }
      }

      useWorkflowStore.getState().handleEvent(event)

      expect(useWorkflowStore.getState().currentPhase).toBe('design')
      expect(useWorkflowStore.getState().tasks).toHaveLength(1)
    })

    it('should handle phase_changed event', () => {
      useWorkflowStore.getState().setPhases('idea', [
        { phase: 'idea', status: 'current' },
        { phase: 'design', status: 'pending' }
      ])

      const event: WorkflowEvent = {
        type: 'phase_changed',
        phase: 'design',
        previous: 'idea'
      }

      useWorkflowStore.getState().handleEvent(event)

      expect(useWorkflowStore.getState().currentPhase).toBe('design')
    })

    it('should handle task_created event', () => {
      const event: WorkflowEvent = {
        type: 'task_created',
        task: {
          id: 'task-new',
          name: 'New Task',
          phase: 'idea',
          zone: 'test',
          status: 'pending',
          persona: 'dev',
          dependencies: [],
          created_at: Date.now(),
          updated_at: Date.now()
        }
      }

      useWorkflowStore.getState().handleEvent(event)

      expect(useWorkflowStore.getState().tasks).toHaveLength(1)
      expect(useWorkflowStore.getState().tasks[0].id).toBe('task-new')
    })

    it('should handle task_updated event', () => {
      useWorkflowStore.getState().addTask({
        id: 'task-1',
        name: 'Test',
        phase: 'idea',
        zone: 'test',
        status: 'pending',
        persona: 'dev',
        dependencies: [],
        created_at: Date.now(),
        updated_at: Date.now()
      })

      const event: WorkflowEvent = {
        type: 'task_updated',
        task_id: 'task-1',
        status: 'done',
        previous: 'pending'
      }

      useWorkflowStore.getState().handleEvent(event)

      expect(useWorkflowStore.getState().tasks[0].status).toBe('done')
    })

    it('should handle checkpoint_created event', () => {
      const event: WorkflowEvent = {
        type: 'checkpoint_created',
        checkpoint_id: 'cp-new',
        phase: 'idea'
      }

      useWorkflowStore.getState().handleEvent(event)

      expect(useWorkflowStore.getState().checkpoints).toHaveLength(1)
      expect(useWorkflowStore.getState().checkpoints[0].id).toBe('cp-new')
    })
  })

  describe('Loading and error state', () => {
    it('should set loading state', () => {
      useWorkflowStore.getState().setLoading(true)
      expect(useWorkflowStore.getState().loading).toBe(true)

      useWorkflowStore.getState().setLoading(false)
      expect(useWorkflowStore.getState().loading).toBe(false)
    })

    it('should set error state', () => {
      useWorkflowStore.getState().setError('Test error')
      expect(useWorkflowStore.getState().error).toBe('Test error')

      useWorkflowStore.getState().setError(null)
      expect(useWorkflowStore.getState().error).toBeNull()
    })
  })
})
