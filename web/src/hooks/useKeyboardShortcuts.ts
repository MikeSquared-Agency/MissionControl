import { useEffect, useCallback } from 'react'
import { useStore } from '../stores/useStore'

interface KeyboardShortcutsOptions {
  onSpawnAgent: () => void
  onNewZone: () => void
  onOpenSettings: () => void
  onCloseModal: () => void
  onFocusChatInput?: () => void
  onKillAgent?: () => void
}

export function useKeyboardShortcuts({
  onSpawnAgent,
  onNewZone,
  onOpenSettings,
  onCloseModal,
  onFocusChatInput,
  onKillAgent
}: KeyboardShortcutsOptions) {
  const agents = useStore((s) => s.agents)
  const zones = useStore((s) => s.zones)
  const selectedAgentId = useStore((s) => s.selectedAgentId)
  const selectAgent = useStore((s) => s.selectAgent)
  const kingMode = useStore((s) => s.kingMode)
  const setKingMode = useStore((s) => s.setKingMode)

  // Get flattened list of agents in zone order for navigation
  const getOrderedAgents = useCallback(() => {
    const ordered: string[] = []
    for (const zone of zones) {
      const zoneAgents = agents.filter((a) => a.zone === zone.id)
      ordered.push(...zoneAgents.map((a) => a.id))
    }
    // Add any agents not in a known zone
    const orphans = agents.filter((a) => !zones.some((z) => z.id === a.zone))
    ordered.push(...orphans.map((a) => a.id))
    return ordered
  }, [agents, zones])

  // Select next agent
  const selectNextAgent = useCallback(() => {
    const ordered = getOrderedAgents()
    if (ordered.length === 0) return

    if (!selectedAgentId) {
      selectAgent(ordered[0])
      return
    }

    const currentIndex = ordered.indexOf(selectedAgentId)
    const nextIndex = (currentIndex + 1) % ordered.length
    selectAgent(ordered[nextIndex])
  }, [getOrderedAgents, selectedAgentId, selectAgent])

  // Select previous agent
  const selectPrevAgent = useCallback(() => {
    const ordered = getOrderedAgents()
    if (ordered.length === 0) return

    if (!selectedAgentId) {
      selectAgent(ordered[ordered.length - 1])
      return
    }

    const currentIndex = ordered.indexOf(selectedAgentId)
    const prevIndex = currentIndex <= 0 ? ordered.length - 1 : currentIndex - 1
    selectAgent(ordered[prevIndex])
  }, [getOrderedAgents, selectedAgentId, selectAgent])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Don't trigger shortcuts when typing in inputs (except for Escape)
      const target = e.target as HTMLElement
      const isInput = target.tagName === 'INPUT' ||
                      target.tagName === 'TEXTAREA' ||
                      target.isContentEditable

      // Escape always works
      if (e.key === 'Escape') {
        onCloseModal()
        return
      }

      // Other shortcuts don't work in inputs
      if (isInput) return

      const isMeta = e.metaKey || e.ctrlKey
      const isShift = e.shiftKey

      // ⌘N or Ctrl+N — Spawn agent (not in King mode)
      if (isMeta && !isShift && e.key === 'n') {
        if (!kingMode) {
          e.preventDefault()
          onSpawnAgent()
        }
        return
      }

      // ⌘⇧N or Ctrl+Shift+N — New zone
      if (isMeta && isShift && e.key === 'N') {
        e.preventDefault()
        onNewZone()
        return
      }

      // ⌘K or Ctrl+K — Kill selected agent
      if (isMeta && !isShift && e.key === 'k') {
        e.preventDefault()
        onKillAgent?.()
        return
      }

      // ⌘⇧K or Ctrl+Shift+K — Toggle King mode
      if (isMeta && isShift && e.key === 'K') {
        e.preventDefault()
        setKingMode(!kingMode)
        return
      }

      // ⌘, or Ctrl+, — Settings
      if (isMeta && e.key === ',') {
        e.preventDefault()
        onOpenSettings()
        return
      }

      // ⌘/ or Ctrl+/ — Focus chat input
      if (isMeta && e.key === '/') {
        e.preventDefault()
        onFocusChatInput?.()
        return
      }

      // Arrow keys for agent navigation (without modifier)
      if (!isMeta && !isShift && !e.altKey) {
        if (e.key === 'ArrowDown' || e.key === 'j') {
          e.preventDefault()
          selectNextAgent()
          return
        }

        if (e.key === 'ArrowUp' || e.key === 'k') {
          e.preventDefault()
          selectPrevAgent()
          return
        }
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [
    onSpawnAgent,
    onNewZone,
    onOpenSettings,
    onCloseModal,
    onFocusChatInput,
    onKillAgent,
    kingMode,
    setKingMode,
    selectNextAgent,
    selectPrevAgent
  ])
}

// Export shortcut descriptions for settings panel
export const KEYBOARD_SHORTCUTS = [
  { key: '⌘N', action: 'Spawn agent' },
  { key: '⌘K', action: 'Kill selected agent' },
  { key: '↓ / j', action: 'Next agent' },
  { key: '↑ / k', action: 'Previous agent' },
  { key: '⌘⇧K', action: 'Toggle King mode' },
  { key: '⌘⇧N', action: 'New zone' },
  { key: '⌘,', action: 'Settings' },
  { key: '⌘/', action: 'Focus chat input' },
  { key: 'Esc', action: 'Close modal' }
]
