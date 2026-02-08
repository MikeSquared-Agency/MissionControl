import { useState, useEffect, useCallback } from 'react'
import { useWebSocket } from './hooks/useWebSocket'
import { useKeyboardShortcuts } from './hooks/useKeyboardShortcuts'
import { useStore, fetchAgents, fetchZones, killAgent, respondToAttention } from './stores/useStore'
import { useProjectStore, useCurrentProject, fetchProjects } from './stores/useProjectStore'
import { Header } from './components/Header'
import { Sidebar } from './components/Sidebar'
import { AgentPanel } from './components/AgentPanel'
import { AttentionBar } from './components/AttentionBar'
import { SpawnDialog } from './components/SpawnDialog'
import { ZoneDialog } from './components/ZoneDialog'
import { SettingsPanel } from './components/SettingsPanel'
import { ConfirmDialog } from './components/ConfirmDialog'
import { MoveAgentDialog } from './components/MoveAgentDialog'
import { ContextShareDialog } from './components/ContextShareDialog'
import { SplitZoneDialog } from './components/SplitZoneDialog'
import { MergeZoneDialog } from './components/MergeZoneDialog'
import { KingPanel } from './components/KingPanel'
import { ProjectWizard } from './components/ProjectWizard'
import { ToastContainer } from './components/Toast'
import { toast } from './stores/useToast'
import { StageView } from './domains/workflow/StageView'
import { TokenUsage } from './domains/knowledge/TokenUsage'
import { GateApproval } from './domains/strategy/GateApproval'
import { FindingsViewer } from './components/FindingsViewer'
import type { Zone, Agent } from './types'

type ViewMode = 'agents' | 'workflow' | 'tokens' | 'gates' | 'findings'

function App() {
  // View mode for v4 panels
  const [viewMode, setViewMode] = useState<ViewMode>('agents')

  // Dialog state
  const [spawnOpen, setSpawnOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)

  // Zone dialogs
  const [zoneDialogOpen, setZoneDialogOpen] = useState(false)
  const [zoneDialogMode, setZoneDialogMode] = useState<'create' | 'edit' | 'duplicate'>('create')
  const [selectedZone, setSelectedZone] = useState<Zone | null>(null)
  const [splitZoneOpen, setSplitZoneOpen] = useState(false)
  const [mergeZoneOpen, setMergeZoneOpen] = useState(false)

  // Agent dialogs
  const [moveAgentOpen, setMoveAgentOpen] = useState(false)
  const [contextShareOpen, setContextShareOpen] = useState(false)
  const [agentForDialog, setAgentForDialog] = useState<Agent | null>(null)

  // Confirm dialog
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [confirmProps, setConfirmProps] = useState({
    title: '',
    message: '',
    onConfirm: () => {},
    danger: false
  })

  // Initialize WebSocket connection
  useWebSocket()

  const setAgents = useStore((s) => s.setAgents)
  const setZones = useStore((s) => s.setZones)
  const kingMode = useStore((s) => s.kingMode)
  const setKingMode = useStore((s) => s.setKingMode)
  const selectAgent = useStore((s) => s.selectAgent)
  const selectedAgentId = useStore((s) => s.selectedAgentId)
  const agents = useStore((s) => s.agents)

  // Project state
  const currentProject = useCurrentProject()
  const setProjects = useProjectStore((s) => s.setProjects)
  const openWizard = useProjectStore((s) => s.openWizard)

  // Fetch initial data on mount
  useEffect(() => {
    fetchAgents()
      .then(setAgents)
      .catch((err) => console.error('Failed to fetch agents:', err))

    fetchZones()
      .then(setZones)
      .catch((err) => console.error('Failed to fetch zones:', err))

    fetchProjects()
      .then(setProjects)
      .catch((err) => console.error('Failed to fetch projects:', err))
  }, [setAgents, setZones, setProjects])

  // Auto-open wizard if no project selected
  useEffect(() => {
    if (currentProject === null) {
      openWizard()
    }
  }, [currentProject, openWizard])

  // Close any open modal
  const closeAllModals = useCallback(() => {
    setSpawnOpen(false)
    setSettingsOpen(false)
    setZoneDialogOpen(false)
    setSplitZoneOpen(false)
    setMergeZoneOpen(false)
    setMoveAgentOpen(false)
    setContextShareOpen(false)
    setConfirmOpen(false)
  }, [])

  // Kill selected agent
  const handleKillAgent = useCallback(() => {
    if (!selectedAgentId) return

    const agent = agents.find((a) => a.id === selectedAgentId)
    if (!agent || agent.status === 'stopped') return

    setConfirmProps({
      title: 'Kill Agent',
      message: `Are you sure you want to kill "${agent.name}"? This cannot be undone.`,
      danger: true,
      onConfirm: async () => {
        try {
          await killAgent(selectedAgentId)
          toast.success(`Agent "${agent.name}" killed`)
          setConfirmOpen(false)
        } catch (err) {
          console.error('Failed to kill agent:', err)
          toast.error('Failed to kill agent')
        }
      }
    })
    setConfirmOpen(true)
  }, [selectedAgentId, agents])

  // Zone context menu actions
  const handleZoneEdit = useCallback((zone: Zone) => {
    setSelectedZone(zone)
    setZoneDialogMode('edit')
    setZoneDialogOpen(true)
  }, [])

  const handleZoneDuplicate = useCallback((zone: Zone) => {
    setSelectedZone(zone)
    setZoneDialogMode('duplicate')
    setZoneDialogOpen(true)
  }, [])

  const handleZoneSplit = useCallback((zone: Zone) => {
    setSelectedZone(zone)
    setSplitZoneOpen(true)
  }, [])

  const handleZoneMerge = useCallback((zone: Zone) => {
    setSelectedZone(zone)
    setMergeZoneOpen(true)
  }, [])

  // Agent context menu actions
  const handleAgentMove = useCallback((agent: Agent) => {
    setAgentForDialog(agent)
    setMoveAgentOpen(true)
  }, [])

  const handleAgentShare = useCallback((agent: Agent) => {
    setAgentForDialog(agent)
    setContextShareOpen(true)
  }, [])

  // Open new zone dialog
  const handleNewZone = useCallback(() => {
    setSelectedZone(null)
    setZoneDialogMode('create')
    setZoneDialogOpen(true)
  }, [])

  // Handle attention response
  const handleAttentionRespond = useCallback(async (agentId: string, response: string) => {
    try {
      await respondToAttention(agentId, response)
    } catch (err) {
      console.error('Failed to respond to attention:', err)
    }
  }, [])

  // Focus chat input
  const handleFocusChatInput = useCallback(() => {
    // Find the chat input in the DOM
    const input = document.querySelector('[data-chat-input]') as HTMLTextAreaElement
    input?.focus()
  }, [])

  // Keyboard shortcuts
  useKeyboardShortcuts({
    onSpawnAgent: () => !kingMode && setSpawnOpen(true),
    onNewZone: handleNewZone,
    onOpenSettings: () => setSettingsOpen(true),
    onCloseModal: closeAllModals,
    onFocusChatInput: handleFocusChatInput,
    onKillAgent: handleKillAgent
  })

  return (
    <div className="h-screen flex flex-col bg-gray-950 text-gray-100">
      {/* Header */}
      <Header
        onSpawnClick={() => setSpawnOpen(true)}
        onNewZoneClick={handleNewZone}
        onSettingsClick={() => setSettingsOpen(true)}
      />

      {/* Attention bar - shows when agents need attention */}
      <AttentionBar onRespond={handleAttentionRespond} />

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar - only show in agents mode */}
        {viewMode === 'agents' && (
          <Sidebar
            onZoneEdit={handleZoneEdit}
            onZoneDuplicate={handleZoneDuplicate}
            onZoneSplit={handleZoneSplit}
            onZoneMerge={handleZoneMerge}
            onNewZone={handleNewZone}
            onAgentMove={handleAgentMove}
            onAgentShare={handleAgentShare}
          />
        )}

        {/* Main panel area */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* View mode tabs */}
          <div className="flex items-center gap-1 px-3 py-2 bg-gray-900 border-b border-gray-800">
            {(['agents', 'workflow', 'tokens', 'gates', 'findings'] as ViewMode[]).map((mode) => (
              <button
                key={mode}
                onClick={() => setViewMode(mode)}
                className={`
                  px-3 py-1.5 text-xs font-medium rounded transition-colors
                  ${viewMode === mode
                    ? 'bg-blue-600 text-white'
                    : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800'
                  }
                `}
              >
                {mode === 'agents' && '🤖 Agents'}
                {mode === 'workflow' && '📋 Workflow'}
                {mode === 'tokens' && '🪙 Tokens'}
                {mode === 'gates' && '🚦 Gates'}
                {mode === 'findings' && '📝 Findings'}
              </button>
            ))}
          </div>

          {/* Panel content */}
          <div className="flex-1 overflow-hidden">
            {kingMode ? (
              <KingPanel
                onExit={() => setKingMode(false)}
                onAgentClick={(agentId) => {
                  setKingMode(false)
                  selectAgent(agentId)
                  setViewMode('agents')
                }}
              />
            ) : viewMode === 'agents' ? (
              <AgentPanel />
            ) : viewMode === 'workflow' ? (
              <StageView />
            ) : viewMode === 'tokens' ? (
              <TokenUsage />
            ) : viewMode === 'gates' ? (
              <GateApproval />
            ) : viewMode === 'findings' ? (
              <div className="p-4 overflow-y-auto h-full">
                <FindingsViewer />
              </div>
            ) : null}
          </div>
        </div>
      </div>

      {/* Dialogs */}
      <SpawnDialog open={spawnOpen} onClose={() => setSpawnOpen(false)} />

      <ZoneDialog
        open={zoneDialogOpen}
        onClose={() => setZoneDialogOpen(false)}
        mode={zoneDialogMode}
        zone={selectedZone ?? undefined}
      />

      <SettingsPanel
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
      />

      <SplitZoneDialog
        open={splitZoneOpen}
        onClose={() => setSplitZoneOpen(false)}
        zone={selectedZone}
      />

      <MergeZoneDialog
        open={mergeZoneOpen}
        onClose={() => setMergeZoneOpen(false)}
        zone={selectedZone}
      />

      <MoveAgentDialog
        open={moveAgentOpen}
        onClose={() => setMoveAgentOpen(false)}
        agent={agentForDialog}
      />

      <ContextShareDialog
        open={contextShareOpen}
        onClose={() => setContextShareOpen(false)}
        sourceAgent={agentForDialog}
      />

      <ConfirmDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={confirmProps.onConfirm}
        title={confirmProps.title}
        message={confirmProps.message}
        danger={confirmProps.danger}
        confirmText="Confirm"
      />

      {/* Project wizard */}
      <ProjectWizard />

      {/* Toast notifications */}
      <ToastContainer />
    </div>
  )
}

export default App
