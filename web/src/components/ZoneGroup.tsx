import { useState, useRef } from 'react'
import { useStore } from '../stores/useStore'
import type { Zone, Agent } from '../types'
import { AgentCard } from './AgentCard'
import { ContextMenu } from './ContextMenu'

interface ZoneGroupProps {
  zone: Zone
  onZoneEdit?: (zone: Zone) => void
  onZoneDuplicate?: (zone: Zone) => void
  onZoneSplit?: (zone: Zone) => void
  onZoneMerge?: (zone: Zone) => void
  onAgentMove?: (agent: Agent) => void
  onAgentShare?: (agent: Agent) => void
}

export function ZoneGroup({
  zone,
  onZoneEdit,
  onZoneDuplicate,
  onZoneSplit,
  onZoneMerge,
  onAgentMove,
  onAgentShare
}: ZoneGroupProps) {
  const allAgents = useStore((s) => s.agents)
  const agents = allAgents.filter((a) => a.zone === zone.id)
  const collapsed = useStore((s) => s.collapsedZones[zone.id] ?? false)
  const toggleZoneCollapse = useStore((s) => s.toggleZoneCollapse)
  const selectedAgentId = useStore((s) => s.selectedAgentId)
  const selectAgent = useStore((s) => s.selectAgent)

  const [menuOpen, setMenuOpen] = useState(false)
  const [menuPosition, setMenuPosition] = useState({ x: 0, y: 0 })
  const menuButtonRef = useRef<HTMLButtonElement>(null)

  const workingCount = agents.filter((a) => a.status === 'working').length
  const waitingCount = agents.filter((a) => a.attention !== null || a.status === 'waiting').length
  const totalCost = agents.reduce((sum, a) => sum + a.cost, 0)

  return (
    <div className="border-b border-gray-800/50 last:border-b-0">
      {/* Zone header */}
      <div
        onClick={() => toggleZoneCollapse(zone.id)}
        className="w-full flex items-center gap-2 px-3 py-2 hover:bg-gray-800/50 transition-colors group cursor-pointer"
      >
        {/* Collapse arrow */}
        <svg
          className={`w-3 h-3 text-gray-500 transition-transform ${collapsed ? '' : 'rotate-90'}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
        </svg>

        {/* Zone color dot */}
        <span
          className="w-2 h-2 rounded-full flex-shrink-0"
          style={{ backgroundColor: zone.color }}
        />

        {/* Zone name */}
        <span className="text-xs font-medium text-gray-300 flex-1 text-left truncate">
          {zone.name}
        </span>

        {/* Stats */}
        <div className="flex items-center gap-2 text-[10px] text-gray-500">
          {/* Agent count */}
          <span>{agents.length}</span>

          {/* Working count */}
          {workingCount > 0 && (
            <span className="text-green-500">{workingCount} working</span>
          )}

          {/* Waiting count with pulse */}
          {waitingCount > 0 && (
            <span className="text-amber-500 animate-pulse">{waitingCount} waiting</span>
          )}

          {/* Cost */}
          {totalCost > 0 && (
            <span className="font-mono">${totalCost.toFixed(2)}</span>
          )}
        </div>

        {/* Zone menu button */}
        <button
          ref={menuButtonRef}
          onClick={(e) => {
            e.stopPropagation()
            const rect = e.currentTarget.getBoundingClientRect()
            setMenuPosition({ x: rect.right, y: rect.bottom })
            setMenuOpen(true)
          }}
          className="p-1 text-gray-600 hover:text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity rounded"
          aria-label="Zone menu"
        >
          <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
            <circle cx="12" cy="6" r="2" />
            <circle cx="12" cy="12" r="2" />
            <circle cx="12" cy="18" r="2" />
          </svg>
        </button>
      </div>

      {/* Zone context menu */}
      <ContextMenu
        open={menuOpen}
        onClose={() => setMenuOpen(false)}
        position={menuPosition}
        items={[
          {
            label: 'Edit Zone',
            onClick: () => {
              setMenuOpen(false)
              onZoneEdit?.(zone)
            }
          },
          {
            label: 'Duplicate Zone',
            onClick: () => {
              setMenuOpen(false)
              onZoneDuplicate?.(zone)
            }
          },
          { divider: true },
          {
            label: 'Split Zone',
            onClick: () => {
              setMenuOpen(false)
              onZoneSplit?.(zone)
            },
            disabled: agents.length < 2
          },
          {
            label: 'Merge into Another',
            onClick: () => {
              setMenuOpen(false)
              onZoneMerge?.(zone)
            }
          }
        ]}
      />

      {/* Agent list */}
      {!collapsed && (
        <div className="pb-1">
          {agents.length === 0 ? (
            <div className="px-3 py-2 text-[11px] text-gray-600 italic">
              No agents in this zone
            </div>
          ) : (
            agents.map((agent) => (
              <AgentCard
                key={agent.id}
                agent={agent}
                selected={agent.id === selectedAgentId}
                onSelect={() => selectAgent(agent.id)}
                onMove={() => onAgentMove?.(agent)}
                onShare={() => onAgentShare?.(agent)}
              />
            ))
          )}
        </div>
      )}
    </div>
  )
}
