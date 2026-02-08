import { useState, useCallback } from 'react'
import { Modal } from './Modal'
import { useStore, fetchPersonaPrompt, updatePersonaPrompt } from '../stores/useStore'
import { useProjectStore } from '../stores/useProjectStore'
import { KEYBOARD_SHORTCUTS } from '../hooks/useKeyboardShortcuts'
import type { Persona } from '../types'
import type { Stage } from '../types/workflow'

interface SettingsPanelProps {
  open: boolean
  onClose: () => void
}

type Tab = 'general' | 'personas' | 'shortcuts'

export function SettingsPanel({ open, onClose }: SettingsPanelProps) {
  const [activeTab, setActiveTab] = useState<Tab>('general')

  return (
    <Modal open={open} onClose={onClose} title="Settings" width="lg">
      <div className="flex gap-4 min-h-[400px]">
        {/* Tab navigation */}
        <div className="w-32 flex-shrink-0 border-r border-gray-800 pr-4">
          <nav className="space-y-1">
            <TabButton
              active={activeTab === 'general'}
              onClick={() => setActiveTab('general')}
            >
              General
            </TabButton>
            <TabButton
              active={activeTab === 'personas'}
              onClick={() => setActiveTab('personas')}
            >
              Personas
            </TabButton>
            <TabButton
              active={activeTab === 'shortcuts'}
              onClick={() => setActiveTab('shortcuts')}
            >
              Shortcuts
            </TabButton>
          </nav>
        </div>

        {/* Tab content */}
        <div className="flex-1 min-w-0">
          {activeTab === 'general' && <GeneralTab />}
          {activeTab === 'personas' && <PersonasTab />}
          {activeTab === 'shortcuts' && <ShortcutsTab />}
        </div>
      </div>
    </Modal>
  )
}

function TabButton({
  active,
  onClick,
  children
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={onClick}
      className={`w-full px-3 py-1.5 text-xs text-left rounded transition-colors ${
        active
          ? 'bg-gray-800 text-gray-100'
          : 'text-gray-500 hover:text-gray-300 hover:bg-gray-800/50'
      }`}
    >
      {children}
    </button>
  )
}

function GeneralTab() {
  const settings = useStore((s) => s.settings)
  const updateSettings = useStore((s) => s.updateSettings)
  const kingMode = useStore((s) => s.kingMode)
  const setKingMode = useStore((s) => s.setKingMode)

  return (
    <div className="space-y-6">
      <h3 className="text-sm font-medium text-gray-100">General Settings</h3>

      {/* API Key */}
      <div>
        <label className="block text-[11px] text-gray-500 mb-1.5">
          Anthropic API Key
        </label>
        <input
          type="password"
          value={settings.apiKey}
          onChange={(e) => updateSettings({ apiKey: e.target.value })}
          placeholder="sk-ant-..."
          className="w-full px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
        />
        <p className="mt-1 text-[10px] text-gray-600">
          Used for Claude Code agents. Leave empty to use environment variable.
        </p>
      </div>

      {/* Default Working Directory */}
      <div>
        <label className="block text-[11px] text-gray-500 mb-1.5">
          Default Working Directory
        </label>
        <input
          type="text"
          value={settings.defaultWorkingDir}
          onChange={(e) => updateSettings({ defaultWorkingDir: e.target.value })}
          placeholder="/path/to/default/project"
          className="w-full px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 placeholder-gray-600 focus:outline-none focus:border-gray-600"
        />
      </div>

      {/* OpenClaw Mode */}
      <div>
        <label className="flex items-center justify-between">
          <div>
            <span className="text-xs text-gray-300">OpenClaw Mode</span>
            <p className="text-[10px] text-gray-600 mt-0.5">
              Let an AI orchestrator manage your agent team
            </p>
          </div>
          <button
            onClick={() => setKingMode(!kingMode)}
            className={`relative w-10 h-5 rounded-full transition-colors ${
              kingMode ? 'bg-amber-500' : 'bg-gray-700'
            }`}
          >
            <span
              className={`absolute top-0.5 w-4 h-4 bg-white rounded-full transition-transform ${
                kingMode ? 'left-5' : 'left-0.5'
              }`}
            />
          </button>
        </label>
      </div>
    </div>
  )
}

function PersonasTab() {
  const personas = useStore((s) => s.personas)
  const updatePersona = useStore((s) => s.updatePersona)
  const addPersona = useStore((s) => s.addPersona)
  const removePersona = useStore((s) => s.removePersona)
  const currentProject = useProjectStore((s) => s.currentProject)

  const [editing, setEditing] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<Partial<Persona>>({})
  const [promptContent, setPromptContent] = useState<string>('')
  const [promptLoading, setPromptLoading] = useState(false)
  const [promptError, setPromptError] = useState<string | null>(null)
  const [promptSaving, setPromptSaving] = useState(false)
  const [promptDirty, setPromptDirty] = useState(false)

  // Load prompt when editing a builtin persona
  const loadPrompt = useCallback(async (personaId: string) => {
    if (!currentProject) {
      setPromptContent('')
      setPromptError('No project selected')
      return
    }

    setPromptLoading(true)
    setPromptError(null)
    try {
      const content = await fetchPersonaPrompt(currentProject, personaId)
      setPromptContent(content)
      setPromptDirty(false)
    } catch (err) {
      setPromptError(err instanceof Error ? err.message : 'Failed to load prompt')
      setPromptContent('')
    } finally {
      setPromptLoading(false)
    }
  }, [currentProject])

  const startEdit = (persona: Persona) => {
    setEditing(persona.id)
    setEditForm({ ...persona })
    setPromptContent('')
    setPromptError(null)
    setPromptDirty(false)

    // Load prompt for builtin personas
    if (persona.isBuiltin) {
      loadPrompt(persona.id)
    }
  }

  const cancelEdit = () => {
    setEditing(null)
    setEditForm({})
    setPromptContent('')
    setPromptError(null)
    setPromptDirty(false)
  }

  const saveEdit = () => {
    if (!editing || !editForm.name) return

    updatePersona(editing, editForm)
    cancelEdit()
  }

  const savePrompt = async () => {
    if (!editing || !currentProject) return

    setPromptSaving(true)
    try {
      await updatePersonaPrompt(currentProject, editing, promptContent)
      setPromptDirty(false)
    } catch (err) {
      setPromptError(err instanceof Error ? err.message : 'Failed to save prompt')
    } finally {
      setPromptSaving(false)
    }
  }

  const createNew = () => {
    const id = `persona-${Date.now()}`
    const newPersona: Persona = {
      id,
      name: 'New Persona',
      description: 'Description here',
      color: '#6b7280',
      stage: 'implement',
      enabled: true,
      tools: ['read', 'write', 'edit', 'bash', 'grep'],
      skills: [],
      systemPrompt: '',
      isBuiltin: false
    }
    addPersona(newPersona)
    startEdit(newPersona)
  }

  const toggleEnabled = (persona: Persona, e: React.MouseEvent) => {
    e.stopPropagation()
    updatePersona(persona.id, { enabled: !persona.enabled })
  }

  // Group personas by stage
  const stages: Stage[] = ['discovery', 'goal', 'requirements', 'planning', 'design', 'implement', 'verify', 'validate', 'document', 'release']
  const stageLabels: Record<string, string> = {
    discovery: 'Discovery',
    goal: 'Goal',
    requirements: 'Requirements',
    planning: 'Planning',
    design: 'Design',
    implement: 'Implement',
    verify: 'Verify',
    validate: 'Validate',
    document: 'Document',
    release: 'Release'
  }

  if (editing) {
    const isBuiltin = editForm.isBuiltin
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-100">
            {isBuiltin ? 'View Persona' : 'Edit Persona'}
          </h3>
          <button
            onClick={cancelEdit}
            className="text-xs text-gray-500 hover:text-gray-300"
          >
            ← Back to list
          </button>
        </div>

        {/* Name */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Name</label>
          <input
            type="text"
            value={editForm.name || ''}
            onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
            disabled={isBuiltin}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
          />
        </div>

        {/* Stage */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Stage</label>
          <select
            value={editForm.stage || 'implement'}
            onChange={(e) => setEditForm({ ...editForm, stage: e.target.value as Persona['stage'] })}
            disabled={isBuiltin}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {stages.map(s => (
              <option key={s} value={s}>{stageLabels[s]}</option>
            ))}
          </select>
        </div>

        {/* Color */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Color</label>
          <div className="flex gap-2 items-center">
            <input
              type="color"
              value={editForm.color || '#6b7280'}
              onChange={(e) => setEditForm({ ...editForm, color: e.target.value })}
              disabled={isBuiltin}
              className="w-10 h-8 bg-gray-800 border border-gray-700/50 rounded cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            />
            <input
              type="text"
              value={editForm.color || ''}
              onChange={(e) => setEditForm({ ...editForm, color: e.target.value })}
              disabled={isBuiltin}
              placeholder="#hex"
              className="w-24 px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
            />
          </div>
        </div>

        {/* Description */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Description</label>
          <input
            type="text"
            value={editForm.description || ''}
            onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
            disabled={isBuiltin}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
          />
        </div>

        {/* Tools */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Tools</label>
          {isBuiltin ? (
            <div className="flex flex-wrap gap-1.5">
              {(editForm.tools || []).map((tool) => (
                <span
                  key={tool}
                  className="px-2 py-0.5 text-[10px] bg-gray-800 text-gray-400 rounded"
                >
                  {tool}
                </span>
              ))}
            </div>
          ) : (
            <input
              type="text"
              value={(editForm.tools || []).join(', ')}
              onChange={(e) => setEditForm({
                ...editForm,
                tools: e.target.value.split(',').map(t => t.trim()).filter(Boolean)
              })}
              placeholder="read, write, edit, bash, grep"
              className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
            />
          )}
          <p className="mt-1 text-[10px] text-gray-600">
            Available: read, write, edit, bash, bash_readonly, grep, tree, web_search
          </p>
        </div>

        {/* Skills */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Skills</label>
          {isBuiltin ? (
            <div className="flex flex-wrap gap-1.5">
              {(editForm.skills || []).map((skill) => (
                <span
                  key={skill}
                  className="px-2 py-0.5 text-[10px] bg-gray-800 text-gray-400 rounded"
                >
                  {skill}
                </span>
              ))}
            </div>
          ) : (
            <input
              type="text"
              value={(editForm.skills || []).join(', ')}
              onChange={(e) => setEditForm({
                ...editForm,
                skills: e.target.value.split(',').map(s => s.trim()).filter(Boolean)
              })}
              placeholder="code-review, implementation, testing"
              className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
            />
          )}
        </div>

        {/* System Prompt */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-[11px] text-gray-500">
              System Prompt {isBuiltin && <span className="text-gray-600">(.mission/prompts/{editing}.md)</span>}
            </label>
            {isBuiltin && promptDirty && (
              <button
                onClick={savePrompt}
                disabled={promptSaving}
                className="text-[10px] text-blue-400 hover:text-blue-300 disabled:opacity-50"
              >
                {promptSaving ? 'Saving...' : 'Save Changes'}
              </button>
            )}
          </div>
          {isBuiltin ? (
            <>
              {promptLoading ? (
                <div className="w-full h-32 flex items-center justify-center bg-gray-800 border border-gray-700/50 rounded text-gray-500 text-xs">
                  Loading prompt...
                </div>
              ) : promptError && !promptContent ? (
                <div className="w-full h-32 flex flex-col items-center justify-center bg-gray-800 border border-gray-700/50 rounded text-xs">
                  <span className="text-gray-500 mb-2">{promptError}</span>
                  <button
                    onClick={() => loadPrompt(editing!)}
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Retry
                  </button>
                </div>
              ) : (
                <textarea
                  value={promptContent}
                  onChange={(e) => {
                    setPromptContent(e.target.value)
                    setPromptDirty(true)
                  }}
                  rows={10}
                  placeholder={currentProject ? 'Enter prompt content...' : 'Select a project to edit prompts'}
                  disabled={!currentProject}
                  className="w-full px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 resize-y focus:outline-none focus:border-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                />
              )}
              {promptError && promptContent && (
                <p className="mt-1 text-[10px] text-red-400">{promptError}</p>
              )}
            </>
          ) : (
            <textarea
              value={editForm.systemPrompt || ''}
              onChange={(e) => setEditForm({ ...editForm, systemPrompt: e.target.value })}
              rows={6}
              className="w-full px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 resize-none focus:outline-none focus:border-gray-600"
            />
          )}
          {isBuiltin && currentProject && (
            <p className="mt-1 text-[10px] text-gray-600">
              Edit the prompt file at .mission/prompts/{editForm.id}.md
            </p>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between pt-2">
          {!isBuiltin ? (
            <button
              onClick={() => {
                if (confirm('Delete this persona?')) {
                  removePersona(editing)
                  cancelEdit()
                }
              }}
              className="px-3 py-1.5 text-xs text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors"
            >
              Delete Persona
            </button>
          ) : (
            <div />
          )}
          <div className="flex gap-2">
            <button
              onClick={cancelEdit}
              className="px-3 py-1.5 text-xs text-gray-400 bg-gray-800 hover:bg-gray-700 rounded transition-colors"
            >
              {isBuiltin ? 'Close' : 'Cancel'}
            </button>
            {!isBuiltin && (
              <button
                onClick={saveEdit}
                className="px-3 py-1.5 text-xs text-white bg-blue-600 hover:bg-blue-500 rounded transition-colors"
              >
                Save
              </button>
            )}
          </div>
        </div>
      </div>
    )
  }

  // Group personas by stage for display
  const builtinPersonas = personas.filter(p => p.isBuiltin)
  const customPersonas = personas.filter(p => !p.isBuiltin)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-100">Workflow Personas</h3>
      </div>

      <p className="text-[11px] text-gray-500">
        Toggle personas on/off to control which workers are available in your workflow.
      </p>

      {/* Builtin personas grouped by stage */}
      <div className="space-y-3">
        {stages.map(stage => {
          const stagePersonas = builtinPersonas.filter(p => p.stage === stage)
          if (stagePersonas.length === 0) return null
          return (
            <div key={stage}>
              <div className="text-[10px] text-gray-600 uppercase tracking-wider mb-1.5">
                {stageLabels[stage]}
              </div>
              <div className="space-y-1">
                {stagePersonas.map((persona) => (
                  <div
                    key={persona.id}
                    className={`flex items-center gap-3 p-2.5 rounded transition-colors ${
                      persona.enabled
                        ? 'bg-gray-800/50 hover:bg-gray-800'
                        : 'bg-gray-900/30 opacity-60'
                    }`}
                  >
                    {/* Enable/Disable Toggle */}
                    <button
                      onClick={(e) => toggleEnabled(persona, e)}
                      className={`relative w-8 h-4 rounded-full transition-colors flex-shrink-0 ${
                        persona.enabled ? 'bg-green-500' : 'bg-gray-700'
                      }`}
                    >
                      <span
                        className={`absolute top-0.5 w-3 h-3 bg-white rounded-full transition-transform ${
                          persona.enabled ? 'left-4' : 'left-0.5'
                        }`}
                      />
                    </button>

                    {/* Color dot */}
                    <span
                      className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                      style={{ backgroundColor: persona.color }}
                    />

                    {/* Info */}
                    <button
                      onClick={() => startEdit(persona)}
                      className="flex-1 min-w-0 text-left"
                    >
                      <p className="text-sm text-gray-200">{persona.name}</p>
                      <p className="text-[10px] text-gray-500 truncate">{persona.description}</p>
                    </button>

                    {/* Tools count */}
                    <span className="text-[10px] text-gray-600">
                      {persona.tools.length} tools
                    </span>

                    {/* View button */}
                    <button
                      onClick={() => startEdit(persona)}
                      className="text-[10px] text-gray-600 hover:text-gray-400 px-2"
                    >
                      View
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )
        })}
      </div>

      {/* Custom personas section */}
      {customPersonas.length > 0 && (
        <div className="pt-2 border-t border-gray-800">
          <div className="text-[10px] text-gray-600 uppercase tracking-wider mb-1.5">
            Custom
          </div>
          <div className="space-y-1">
            {customPersonas.map((persona) => (
              <div
                key={persona.id}
                className={`flex items-center gap-3 p-2.5 rounded transition-colors ${
                  persona.enabled
                    ? 'bg-gray-800/50 hover:bg-gray-800'
                    : 'bg-gray-900/30 opacity-60'
                }`}
              >
                <button
                  onClick={(e) => toggleEnabled(persona, e)}
                  className={`relative w-8 h-4 rounded-full transition-colors flex-shrink-0 ${
                    persona.enabled ? 'bg-green-500' : 'bg-gray-700'
                  }`}
                >
                  <span
                    className={`absolute top-0.5 w-3 h-3 bg-white rounded-full transition-transform ${
                      persona.enabled ? 'left-4' : 'left-0.5'
                    }`}
                  />
                </button>
                <span
                  className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                  style={{ backgroundColor: persona.color }}
                />
                <button
                  onClick={() => startEdit(persona)}
                  className="flex-1 min-w-0 text-left"
                >
                  <p className="text-sm text-gray-200">{persona.name}</p>
                  <p className="text-[10px] text-gray-500 truncate">{persona.description}</p>
                </button>
                <span className="text-[10px] text-gray-600">
                  {persona.tools.length} tools
                </span>
                <button
                  onClick={() => startEdit(persona)}
                  className="text-[10px] text-gray-600 hover:text-gray-400 px-2"
                >
                  Edit
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Add custom persona button */}
      <button
        onClick={createNew}
        className="w-full py-2 text-[11px] text-gray-500 hover:text-gray-300 hover:bg-gray-800/50 rounded border border-dashed border-gray-700 transition-colors"
      >
        + Add Custom Persona
      </button>
    </div>
  )
}

function ShortcutsTab() {
  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-gray-100">Keyboard Shortcuts</h3>

      <div className="grid grid-cols-2 gap-x-8 gap-y-2">
        {KEYBOARD_SHORTCUTS.map((shortcut) => (
          <div key={shortcut.key} className="flex items-center justify-between py-1">
            <span className="text-xs text-gray-400">{shortcut.action}</span>
            <kbd className="px-2 py-0.5 text-[11px] font-mono text-gray-300 bg-gray-800 rounded">
              {shortcut.key}
            </kbd>
          </div>
        ))}
      </div>

      <p className="text-[10px] text-gray-600 pt-4">
        Tip: Use <kbd className="px-1 py-0.5 bg-gray-800 rounded">j</kbd> and{' '}
        <kbd className="px-1 py-0.5 bg-gray-800 rounded">k</kbd> for vim-style navigation
      </p>
    </div>
  )
}
