import { describe, it, expect, beforeEach } from 'vitest'
import { useStore } from './useStore'
import type { Agent, Zone, ConversationMessage, KingMessage } from '../types'

describe('useStore', () => {
  beforeEach(() => {
    // Reset store to initial state before each test
    useStore.setState({
      connectionStatus: 'disconnected',
      agents: [],
      zones: [{ id: 'default', name: 'Default', color: '#6b7280', workingDir: '' }],
      personas: useStore.getState().personas, // Keep default personas
      selectedAgentId: null,
      collapsedZones: {},
      kingMode: false,
      kingConversation: [],
      settings: {
        apiKey: '',
        defaultWorkingDir: ''
      }
    })
  })

  describe('Agent actions', () => {
    const testAgent: Agent = {
      id: 'agent-1',
      name: 'Test Agent',
      type: 'claude-code',
      persona: null,
      status: 'working',
      tokens: 100,
      cost: 0.01,
      task: 'Test task',
      zone: 'default',
      workingDir: '/tmp',
      attention: null,
      findings: [],
      conversation: [],
      created_at: new Date().toISOString()
    }

    it('should add an agent', () => {
      useStore.getState().addAgent(testAgent)
      expect(useStore.getState().agents).toHaveLength(1)
      expect(useStore.getState().agents[0].id).toBe('agent-1')
    })

    it('should set multiple agents', () => {
      const agents = [testAgent, { ...testAgent, id: 'agent-2', name: 'Agent 2' }]
      useStore.getState().setAgents(agents)
      expect(useStore.getState().agents).toHaveLength(2)
    })

    it('should update an agent', () => {
      useStore.getState().addAgent(testAgent)
      useStore.getState().updateAgent('agent-1', { status: 'idle', tokens: 200 })

      const agent = useStore.getState().agents[0]
      expect(agent.status).toBe('idle')
      expect(agent.tokens).toBe(200)
    })

    it('should remove an agent', () => {
      useStore.getState().addAgent(testAgent)
      useStore.getState().removeAgent('agent-1')
      expect(useStore.getState().agents).toHaveLength(0)
    })

    it('should clear selected agent when removing selected', () => {
      useStore.getState().addAgent(testAgent)
      useStore.getState().selectAgent('agent-1')
      useStore.getState().removeAgent('agent-1')
      expect(useStore.getState().selectedAgentId).toBeNull()
    })
  })

  describe('Zone actions', () => {
    const testZone: Zone = {
      id: 'zone-1',
      name: 'Test Zone',
      color: '#22c55e',
      workingDir: '/tmp'
    }

    it('should add a zone', () => {
      useStore.getState().addZone(testZone)
      expect(useStore.getState().zones).toHaveLength(2) // default + new
    })

    it('should update a zone', () => {
      useStore.getState().addZone(testZone)
      useStore.getState().updateZone('zone-1', { name: 'Updated Zone' })

      const zone = useStore.getState().zones.find(z => z.id === 'zone-1')
      expect(zone?.name).toBe('Updated Zone')
    })

    it('should remove a zone', () => {
      useStore.getState().addZone(testZone)
      useStore.getState().removeZone('zone-1')
      expect(useStore.getState().zones).toHaveLength(1) // only default
    })

    it('should clean up collapsed state when removing zone', () => {
      useStore.getState().addZone(testZone)
      useStore.getState().toggleZoneCollapse('zone-1')
      expect(useStore.getState().collapsedZones['zone-1']).toBe(true)

      useStore.getState().removeZone('zone-1')
      expect(useStore.getState().collapsedZones['zone-1']).toBeUndefined()
    })
  })

  describe('Selection actions', () => {
    it('should select an agent', () => {
      useStore.getState().selectAgent('agent-1')
      expect(useStore.getState().selectedAgentId).toBe('agent-1')
    })

    it('should clear selection', () => {
      useStore.getState().selectAgent('agent-1')
      useStore.getState().selectAgent(null)
      expect(useStore.getState().selectedAgentId).toBeNull()
    })

    it('should toggle zone collapse', () => {
      useStore.getState().toggleZoneCollapse('default')
      expect(useStore.getState().collapsedZones['default']).toBe(true)

      useStore.getState().toggleZoneCollapse('default')
      expect(useStore.getState().collapsedZones['default']).toBe(false)
    })
  })

  describe('King mode actions', () => {
    it('should toggle king mode', () => {
      useStore.getState().setKingMode(true)
      expect(useStore.getState().kingMode).toBe(true)

      useStore.getState().setKingMode(false)
      expect(useStore.getState().kingMode).toBe(false)
    })

    it('should clear selected agent when entering king mode', () => {
      useStore.getState().selectAgent('agent-1')
      useStore.getState().setKingMode(true)
      expect(useStore.getState().selectedAgentId).toBeNull()
    })

    it('should add king message', () => {
      const message: KingMessage = {
        role: 'user',
        content: 'Hello King',
        timestamp: Date.now()
      }
      useStore.getState().addKingMessage(message)
      expect(useStore.getState().kingConversation).toHaveLength(1)
    })

    it('should clear king conversation', () => {
      const message: KingMessage = {
        role: 'user',
        content: 'Hello King',
        timestamp: Date.now()
      }
      useStore.getState().addKingMessage(message)
      useStore.getState().clearKingConversation()
      expect(useStore.getState().kingConversation).toHaveLength(0)
    })
  })

  describe('Settings actions', () => {
    it('should update settings', () => {
      useStore.getState().updateSettings({ apiKey: 'test-key' })
      expect(useStore.getState().settings.apiKey).toBe('test-key')
    })

    it('should preserve other settings when updating', () => {
      useStore.getState().updateSettings({ defaultWorkingDir: '/home' })
      useStore.getState().updateSettings({ apiKey: 'test-key' })

      expect(useStore.getState().settings.defaultWorkingDir).toBe('/home')
      expect(useStore.getState().settings.apiKey).toBe('test-key')
    })
  })

  describe('Connection status', () => {
    it('should set connection status', () => {
      useStore.getState().setConnectionStatus('connected')
      expect(useStore.getState().connectionStatus).toBe('connected')

      useStore.getState().setConnectionStatus('disconnected')
      expect(useStore.getState().connectionStatus).toBe('disconnected')
    })
  })

  describe('Conversation actions', () => {
    const testAgent: Agent = {
      id: 'agent-1',
      name: 'Test Agent',
      type: 'claude-code',
      persona: null,
      status: 'working',
      tokens: 0,
      cost: 0,
      task: 'Test',
      zone: 'default',
      workingDir: '',
      attention: null,
      findings: [],
      conversation: [],
      created_at: new Date().toISOString()
    }

    it('should add a message to agent conversation', () => {
      useStore.getState().addAgent(testAgent)

      const message: ConversationMessage = {
        role: 'user',
        content: 'Hello',
        timestamp: Date.now()
      }
      useStore.getState().addMessage('agent-1', message)

      const agent = useStore.getState().agents[0]
      expect(agent.conversation).toHaveLength(1)
      expect(agent.conversation[0].content).toBe('Hello')
    })

    it('should add a finding to agent', () => {
      useStore.getState().addAgent(testAgent)
      useStore.getState().addFinding('agent-1', 'Found a bug')

      const agent = useStore.getState().agents[0]
      expect(agent.findings).toContain('Found a bug')
    })

    it('should clear agent conversation', () => {
      useStore.getState().addAgent(testAgent)
      useStore.getState().addMessage('agent-1', {
        role: 'user',
        content: 'Hello',
        timestamp: Date.now()
      })
      useStore.getState().addFinding('agent-1', 'Finding')

      useStore.getState().clearConversation('agent-1')

      const agent = useStore.getState().agents[0]
      expect(agent.conversation).toHaveLength(0)
      expect(agent.findings).toHaveLength(0)
    })

    it('should set agent attention', () => {
      useStore.getState().addAgent(testAgent)

      useStore.getState().setAgentAttention('agent-1', {
        type: 'question',
        message: 'Do you want to continue?',
        since: Date.now()
      })

      const agent = useStore.getState().agents[0]
      expect(agent.attention).not.toBeNull()
      expect(agent.attention?.type).toBe('question')
      expect(agent.status).toBe('waiting')
    })

    it('should clear agent attention', () => {
      useStore.getState().addAgent(testAgent)
      useStore.getState().setAgentAttention('agent-1', {
        type: 'question',
        message: 'Question?',
        since: Date.now()
      })

      useStore.getState().setAgentAttention('agent-1', null)

      const agent = useStore.getState().agents[0]
      expect(agent.attention).toBeNull()
    })
  })

  describe('Persona actions', () => {
    it('should have default personas', () => {
      const personas = useStore.getState().personas
      expect(personas.length).toBeGreaterThan(0)
    })

    it('should add a persona', () => {
      const initialCount = useStore.getState().personas.length
      useStore.getState().addPersona({
        id: 'custom',
        name: 'Custom Persona',
        description: 'A custom persona',
        color: '#ff0000',
        tools: ['read'],
        skills: [],
        systemPrompt: 'You are a custom agent'
      })
      expect(useStore.getState().personas).toHaveLength(initialCount + 1)
    })

    it('should update a persona', () => {
      useStore.getState().addPersona({
        id: 'custom',
        name: 'Custom Persona',
        description: 'A custom persona',
        color: '#ff0000',
        tools: [],
        skills: [],
        systemPrompt: ''
      })

      useStore.getState().updatePersona('custom', { name: 'Updated Persona' })

      const persona = useStore.getState().personas.find(p => p.id === 'custom')
      expect(persona?.name).toBe('Updated Persona')
    })

    it('should remove a persona', () => {
      // First ensure we have a clean state with a known persona
      const customPersona = {
        id: 'to-remove',
        name: 'To Remove Persona',
        description: '',
        color: '#ff0000',
        tools: [],
        skills: [],
        systemPrompt: ''
      }
      useStore.getState().addPersona(customPersona)

      const countBefore = useStore.getState().personas.length
      const hasPersona = useStore.getState().personas.some(p => p.id === 'to-remove')
      expect(hasPersona).toBe(true)

      useStore.getState().removePersona('to-remove')

      const countAfter = useStore.getState().personas.length
      expect(countAfter).toBe(countBefore - 1)

      const stillHasPersona = useStore.getState().personas.some(p => p.id === 'to-remove')
      expect(stillHasPersona).toBe(false)
    })
  })
})

describe('Selectors', () => {
  beforeEach(() => {
    useStore.setState({
      agents: [],
      zones: [{ id: 'default', name: 'Default', color: '#6b7280', workingDir: '' }],
      selectedAgentId: null
    })
  })

  it('useAgentsByZone should filter agents by zone', () => {
    const agents: Agent[] = [
      { id: '1', name: 'A1', type: 'claude-code', persona: null, status: 'working', tokens: 0, cost: 0, task: '', zone: 'default', workingDir: '', attention: null, findings: [], conversation: [], created_at: '' },
      { id: '2', name: 'A2', type: 'claude-code', persona: null, status: 'working', tokens: 0, cost: 0, task: '', zone: 'other', workingDir: '', attention: null, findings: [], conversation: [], created_at: '' },
      { id: '3', name: 'A3', type: 'claude-code', persona: null, status: 'working', tokens: 0, cost: 0, task: '', zone: 'default', workingDir: '', attention: null, findings: [], conversation: [], created_at: '' },
    ]
    useStore.setState({ agents })

    const defaultAgents = useStore.getState().agents.filter(a => a.zone === 'default')
    expect(defaultAgents).toHaveLength(2)
  })

  it('useAgentsNeedingAttention should filter agents with attention', () => {
    const agents: Agent[] = [
      { id: '1', name: 'A1', type: 'claude-code', persona: null, status: 'working', tokens: 0, cost: 0, task: '', zone: 'default', workingDir: '', attention: null, findings: [], conversation: [], created_at: '' },
      { id: '2', name: 'A2', type: 'claude-code', persona: null, status: 'waiting', tokens: 0, cost: 0, task: '', zone: 'default', workingDir: '', attention: { type: 'question', message: 'Q?', since: 0 }, findings: [], conversation: [], created_at: '' },
    ]
    useStore.setState({ agents })

    const needingAttention = useStore.getState().agents.filter(a => a.attention !== null)
    expect(needingAttention).toHaveLength(1)
  })
})
