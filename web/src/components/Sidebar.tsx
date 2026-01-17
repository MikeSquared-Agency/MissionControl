import { useStore } from '../stores/useStore'
import { ZoneGroup } from './ZoneGroup'
import type { Zone, Agent } from '../types'

interface SidebarProps {
  onZoneEdit?: (zone: Zone) => void
  onZoneDuplicate?: (zone: Zone) => void
  onZoneSplit?: (zone: Zone) => void
  onZoneMerge?: (zone: Zone) => void
  onNewZone?: () => void
  onAgentMove?: (agent: Agent) => void
  onAgentShare?: (agent: Agent) => void
}

export function Sidebar({
  onZoneEdit,
  onZoneDuplicate,
  onZoneSplit,
  onZoneMerge,
  onNewZone,
  onAgentMove,
  onAgentShare
}: SidebarProps) {
  const zones = useStore((s) => s.zones)
  const kingMode = useStore((s) => s.kingMode)

  return (
    <aside className="w-72 flex flex-col bg-gray-900 border-r border-gray-800/50">
      {/* King mode indicator */}
      {kingMode && (
        <div className="px-3 py-2 bg-amber-500/10 border-b border-amber-500/20">
          <div className="flex items-center gap-2">
            <span className="text-amber-500">👑</span>
            <span className="text-xs text-amber-400">King is managing agents</span>
          </div>
        </div>
      )}

      {/* Zones list */}
      <div className="flex-1 overflow-y-auto">
        {zones.length === 0 ? (
          <div className="px-4 py-8 text-center">
            <p className="text-sm text-gray-500">No zones created</p>
            <button
              onClick={onNewZone}
              className="mt-2 text-xs text-blue-500 hover:text-blue-400"
            >
              Create a zone
            </button>
          </div>
        ) : (
          zones.map((zone) => (
            <ZoneGroup
              key={zone.id}
              zone={zone}
              onZoneEdit={onZoneEdit}
              onZoneDuplicate={onZoneDuplicate}
              onZoneSplit={onZoneSplit}
              onZoneMerge={onZoneMerge}
              onAgentMove={onAgentMove}
              onAgentShare={onAgentShare}
            />
          ))
        )}
      </div>

      {/* Footer with quick actions */}
      <div className="px-3 py-2 border-t border-gray-800/50">
        <div className="flex items-center justify-between text-[10px] text-gray-600">
          <span>⌘N spawn · ⌘⇧N zone</span>
        </div>
      </div>
    </aside>
  )
}
