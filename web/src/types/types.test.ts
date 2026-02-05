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
    it('should have all 13 workflow personas', () => {
      expect(DEFAULT_PERSONAS.length).toBe(13)
    })

    it('should have required properties on each persona', () => {
      DEFAULT_PERSONAS.forEach(persona => {
        expect(persona.id).toBeDefined()
        expect(persona.name).toBeDefined()
        expect(persona.description).toBeDefined()
        expect(persona.color).toBeDefined()
        expect(persona.stage).toBeDefined()
        expect(typeof persona.enabled).toBe('boolean')
        expect(typeof persona.isBuiltin).toBe('boolean')
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

    it('should have all expected persona IDs', () => {
      const expectedIds = [
        'researcher', 'analyst', 'requirements-engineer', 'architect',
        'designer', 'developer', 'debugger',
        'reviewer', 'security', 'tester', 'qa', 'docs', 'devops'
      ]
      const actualIds = DEFAULT_PERSONAS.map(p => p.id)
      expectedIds.forEach(id => {
        expect(actualIds).toContain(id)
      })
    })

    it('should have personas for each workflow stage', () => {
      const stages = DEFAULT_PERSONAS.map(p => p.stage)
      expect(stages).toContain('discovery')
      expect(stages).toContain('goal')
      expect(stages).toContain('requirements')
      expect(stages).toContain('planning')
      expect(stages).toContain('design')
      expect(stages).toContain('implement')
      expect(stages).toContain('verify')
      expect(stages).toContain('validate')
      expect(stages).toContain('document')
      expect(stages).toContain('release')
    })

    it('should have correct stage assignments', () => {
      const stageMap: Record<string, string> = {
        researcher: 'discovery',
        analyst: 'goal',
        'requirements-engineer': 'requirements',
        architect: 'planning',
        designer: 'design',
        developer: 'implement',
        debugger: 'implement',
        reviewer: 'verify',
        security: 'verify',
        tester: 'verify',
        qa: 'validate',
        docs: 'document',
        devops: 'release'
      }
      DEFAULT_PERSONAS.forEach(persona => {
        expect(persona.stage).toBe(stageMap[persona.id])
      })
    })

    it('should have security, qa, and devops disabled by default', () => {
      const security = DEFAULT_PERSONAS.find(p => p.id === 'security')
      const qa = DEFAULT_PERSONAS.find(p => p.id === 'qa')
      const devops = DEFAULT_PERSONAS.find(p => p.id === 'devops')

      expect(security?.enabled).toBe(false)
      expect(qa?.enabled).toBe(false)
      expect(devops?.enabled).toBe(false)
    })

    it('should mark all default personas as builtin', () => {
      DEFAULT_PERSONAS.forEach(persona => {
        expect(persona.isBuiltin).toBe(true)
      })
    })

    it('should have at least one tool per persona', () => {
      DEFAULT_PERSONAS.forEach(persona => {
        expect(persona.tools.length).toBeGreaterThan(0)
      })
    })

    it('should have at least one skill per persona', () => {
      DEFAULT_PERSONAS.forEach(persona => {
        expect(persona.skills.length).toBeGreaterThan(0)
      })
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
        stage: 'implement',
        enabled: true,
        tools: ['read', 'write', 'bash'],
        skills: ['testing', 'debugging'],
        systemPrompt: 'You are a test agent.',
        isBuiltin: false
      }

      expect(persona.id).toBe('custom')
      expect(persona.stage).toBe('implement')
      expect(persona.enabled).toBe(true)
      expect(persona.isBuiltin).toBe(false)
      expect(persona.tools).toContain('read')
      expect(persona.skills).toContain('testing')
    })

    it('should create a valid builtin persona object', () => {
      const persona: Persona = {
        id: 'developer',
        name: 'Developer',
        description: 'Production code and tests',
        color: '#3b82f6',
        stage: 'implement',
        enabled: true,
        tools: ['read', 'write', 'edit', 'bash'],
        skills: ['implementation', 'testing'],
        systemPrompt: 'You are a Developer.',
        isBuiltin: true
      }

      expect(persona.isBuiltin).toBe(true)
      expect(persona.stage).toBe('implement')
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
