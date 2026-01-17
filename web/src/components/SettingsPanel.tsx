import { useState } from 'react'
import { Modal } from './Modal'
import { useStore } from '../stores/useStore'
import { KEYBOARD_SHORTCUTS } from '../hooks/useKeyboardShortcuts'
import type { Persona } from '../types'

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

      {/* King Mode */}
      <div>
        <label className="flex items-center justify-between">
          <div>
            <span className="text-xs text-gray-300">King Mode</span>
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

  const [editing, setEditing] = useState<string | null>(null)
  const [editForm, setEditForm] = useState<Partial<Persona>>({})

  const startEdit = (persona: Persona) => {
    setEditing(persona.id)
    setEditForm({ ...persona })
  }

  const cancelEdit = () => {
    setEditing(null)
    setEditForm({})
  }

  const saveEdit = () => {
    if (!editing || !editForm.name) return

    updatePersona(editing, editForm)
    cancelEdit()
  }

  const createNew = () => {
    const id = `persona-${Date.now()}`
    const newPersona: Persona = {
      id,
      name: 'New Persona',
      description: 'Description here',
      color: '#6b7280',
      tools: [],
      skills: [],
      systemPrompt: ''
    }
    addPersona(newPersona)
    startEdit(newPersona)
  }

  if (editing) {
    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-medium text-gray-100">Edit Persona</h3>
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
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* Color */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Color</label>
          <input
            type="text"
            value={editForm.color || ''}
            onChange={(e) => setEditForm({ ...editForm, color: e.target.value })}
            placeholder="#hex"
            className="w-32 px-3 py-2 text-sm font-mono bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* Description */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">Description</label>
          <input
            type="text"
            value={editForm.description || ''}
            onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* System Prompt */}
        <div>
          <label className="block text-[11px] text-gray-500 mb-1">System Prompt Addition</label>
          <textarea
            value={editForm.systemPrompt || ''}
            onChange={(e) => setEditForm({ ...editForm, systemPrompt: e.target.value })}
            rows={3}
            className="w-full px-3 py-2 text-sm bg-gray-800 border border-gray-700/50 rounded text-gray-100 resize-none focus:outline-none focus:border-gray-600"
          />
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between pt-2">
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
          <div className="flex gap-2">
            <button
              onClick={cancelEdit}
              className="px-3 py-1.5 text-xs text-gray-400 bg-gray-800 hover:bg-gray-700 rounded transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={saveEdit}
              className="px-3 py-1.5 text-xs text-white bg-blue-600 hover:bg-blue-500 rounded transition-colors"
            >
              Save
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-gray-100">Personas</h3>
        <button
          onClick={createNew}
          className="px-2 py-1 text-[11px] text-blue-400 hover:text-blue-300 hover:bg-blue-500/10 rounded transition-colors"
        >
          + New Persona
        </button>
      </div>

      <div className="space-y-2">
        {personas.map((persona) => (
          <button
            key={persona.id}
            onClick={() => startEdit(persona)}
            className="w-full flex items-center gap-3 p-3 bg-gray-800/50 hover:bg-gray-800 rounded transition-colors text-left"
          >
            <span
              className="w-3 h-3 rounded-full flex-shrink-0"
              style={{ backgroundColor: persona.color }}
            />
            <div className="flex-1 min-w-0">
              <p className="text-sm text-gray-200">{persona.name}</p>
              <p className="text-[10px] text-gray-500 truncate">{persona.description}</p>
            </div>
            <span className="text-[10px] text-gray-600">
              {persona.tools.length} tools
            </span>
          </button>
        ))}
      </div>
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
