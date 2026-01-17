import { describe, it, expect } from 'vitest'
import { DEFAULT_PERSONAS, DEFAULT_ZONE } from '../types'
import type { Agent, Zone, Persona, AgentStatus, AgentType } from '../types'

describe('Types', () => {
  describe('DEFAULT_ZONE', () => {
    it('should have correct default zone properties', () => {
      expect(DEFAULT_ZONE.id).toBe('default')
      expect(DEFAULT_ZONE.name).toBe('Default')
      expect(DEFAULT_ZONE.color).toBeDefined()
      expect(DEFAULT_ZONE.workingDir).toBe('')
    })
  })

  describe('DEFAULT_PERSONAS', () => {
    it('should have at least one default persona', () => {
      expect(DEFAULT_PERSONAS.length).toBeGreaterThan(0)
    })

    it('should have required properties on each persona', () => {
      DEFAULT_PERSONAS.forEach(persona => {
        expect(persona.id).toBeDefined()
        expect(persona.name).toBeDefined()
        expect(persona.description).toBeDefined()
        expect(persona.color).toBeDefined()
        expect(Array.isArray(persona.tools)).toBe(true)
        expect(Array.isArray(persona.skills)).toBe(true)
        expect(persona.systemPrompt).toBeDefined()
      })
    })

    it('should have unique ids for each persona', () => {
      const ids = DEFAULT_PERSONAS.map(p => p.id)
      const uniqueIds = [...new Set(ids)]
      expect(uniqueIds.length).toBe(ids.length)
    })
  })

  describe('Agent type', () => {
    it('should allow valid agent statuses', () => {
      const statuses: AgentStatus[] = ['starting', 'working', 'idle', 'error', 'waiting', 'stopped']
      statuses.forEach(status => {
        expect(['starting', 'working', 'idle', 'error', 'waiting', 'stopped']).toContain(status)
      })
    })

    it('should allow valid agent types', () => {
      const types: AgentType[] = ['python', 'claude-code']
      types.forEach(type => {
        expect(['python', 'claude-code']).toContain(type)
      })
    })

    it('should create a valid agent object', () => {
      const agent: Agent = {
        id: 'test-id',
        name: 'Test Agent',
        type: 'claude-code',
        persona: 'developer',
        status: 'working',
        tokens: 100,
        cost: 0.01,
        task: 'Do something',
        zone: 'default',
        workingDir: '/tmp',
        attention: null,
        findings: ['Found something'],
        conversation: [],
        created_at: '2024-01-01T00:00:00Z'
      }

      expect(agent.id).toBe('test-id')
      expect(agent.type).toBe('claude-code')
      expect(agent.status).toBe('working')
    })
  })

  describe('Zone type', () => {
    it('should create a valid zone object', () => {
      const zone: Zone = {
        id: 'test-zone',
        name: 'Test Zone',
        color: '#22c55e',
        workingDir: '/home/user'
      }

      expect(zone.id).toBe('test-zone')
      expect(zone.color).toMatch(/^#[0-9a-fA-F]{6}$/)
    })
  })

  describe('Persona type', () => {
    it('should create a valid persona object', () => {
      const persona: Persona = {
        id: 'custom',
        name: 'Custom Persona',
        description: 'A custom persona for testing',
        color: '#ff0000',
        tools: ['read', 'write', 'bash'],
        skills: ['testing', 'debugging'],
        systemPrompt: 'You are a test agent.'
      }

      expect(persona.id).toBe('custom')
      expect(persona.tools).toContain('read')
      expect(persona.skills).toContain('testing')
    })
  })

  describe('AttentionRequest type', () => {
    it('should support all attention types', () => {
      const types = ['question', 'permission', 'error', 'complete']
      types.forEach(type => {
        expect(['question', 'permission', 'error', 'complete']).toContain(type)
      })
    })
  })
})
