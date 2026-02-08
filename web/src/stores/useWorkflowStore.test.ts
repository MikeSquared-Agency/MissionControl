import { describe, it, expect, beforeEach } from 'vitest'
import { useWorkflowStore } from './useWorkflowStore'
import type { Task, Stage, WorkflowEvent } from '../types/workflow'

describe('useWorkflowStore', () => {
  beforeEach(() => {
    // Reset store to initial state before each test
    useWorkflowStore.setState({
      currentStage: 'discovery',
      stages: [],
      tasks: [],
      gates: {},
      checkpoints: [],
      loading: false,
      error: null
    })
  })

  describe('Stage actions', () => {
    it('should set stages', () => {
      const stages = [
        { stage: 'discovery' as Stage, status: 'current' as const },
        { stage: 'goal' as Stage, status: 'pending' as const }
      ]
      useWorkflowStore.getState().setStages('discovery', stages)

      expect(useWorkflowStore.getState().currentStage).toBe('discovery')
      expect(useWorkflowStore.getState().stages).toHaveLength(2)
    })
  })

  describe('Task actions', () => {
    const testTask: Task = {
      id: 'task-1',
      name: 'Test Task',
      stage: 'discovery',
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
        id: 'gate-discovery',
        stage: 'discovery' as Stage,
        status: 'closed' as const,
        criteria: [{ description: 'Test criterion', satisfied: false }]
      }
      useWorkflowStore.getState().setGate('discovery', gate)

      expect(useWorkflowStore.getState().gates['discovery']).toBeDefined()
      expect(useWorkflowStore.getState().gates['discovery'].status).toBe('closed')
    })
  })

  describe('Checkpoint actions', () => {
    it('should set checkpoints', () => {
      const checkpoints = [
        { id: 'cp-1', stage: 'discovery' as Stage, created_at: Date.now() }
      ]
      useWorkflowStore.getState().setCheckpoints(checkpoints)

      expect(useWorkflowStore.getState().checkpoints).toHaveLength(1)
    })

    it('should add a checkpoint', () => {
      useWorkflowStore.getState().addCheckpoint({
        id: 'cp-2',
        stage: 'design',
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
          current_stage: 'design',
          stages: [
            { stage: 'discovery', status: 'complete' },
            { stage: 'design', status: 'current' }
          ],
          tasks: [{
            id: 'task-1',
            name: 'Test',
            stage: 'design',
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

      expect(useWorkflowStore.getState().currentStage).toBe('design')
      expect(useWorkflowStore.getState().tasks).toHaveLength(1)
    })

    it('should handle stage_changed event', () => {
      useWorkflowStore.getState().setStages('discovery', [
        { stage: 'discovery', status: 'current' },
        { stage: 'goal', status: 'pending' }
      ])

      const event: WorkflowEvent = {
        type: 'stage_changed',
        stage: 'goal',
        previous: 'discovery'
      }

      useWorkflowStore.getState().handleEvent(event)

      expect(useWorkflowStore.getState().currentStage).toBe('goal')
    })

    it('should handle task_created event', () => {
      const event: WorkflowEvent = {
        type: 'task_created',
        task: {
          id: 'task-new',
          name: 'New Task',
          stage: 'discovery',
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
        stage: 'discovery',
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
        stage: 'discovery'
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
