import { useStore } from '../stores/useStore'
import { useProjectStore, useProjects, useCurrentProject } from '../stores/useProjectStore'
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

  // Project state
  const projects = useProjects()
  const currentProject = useCurrentProject()
  const setCurrentProject = useProjectStore((s) => s.setCurrentProject)
  const openWizard = useProjectStore((s) => s.openWizard)

  // Get current project object
  const currentProjectObj = projects.find((p) => p.path === currentProject)

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

      {/* Projects section */}
      <div className="border-b border-gray-800/50">
        <div className="px-3 py-2 flex items-center justify-between">
          <span className="text-[10px] font-medium text-gray-500 uppercase tracking-wide">
            Projects
          </span>
          <button
            onClick={openWizard}
            className="w-5 h-5 flex items-center justify-center text-gray-500 hover:text-gray-300 hover:bg-gray-800 rounded transition-colors"
            title="New project"
          >
            +
          </button>
        </div>

        {/* Current project */}
        {currentProjectObj && (
          <div className="px-3 pb-2">
            <div className="px-2 py-1.5 bg-gray-800/50 rounded text-sm text-gray-200 truncate">
              {currentProjectObj.name}
            </div>
          </div>
        )}

        {/* Project list (if more than one) */}
        {projects.length > 1 && (
          <div className="px-3 pb-2">
            <div className="space-y-0.5">
              {projects
                .filter((p) => p.path !== currentProject)
                .map((project) => (
                  <button
                    key={project.path}
                    onClick={() => setCurrentProject(project.path)}
                    className="w-full px-2 py-1 text-left text-xs text-gray-500 hover:text-gray-300 hover:bg-gray-800/50 rounded truncate transition-colors"
                    title={project.path}
                  >
                    {project.name}
                  </button>
                ))}
            </div>
          </div>
        )}

        {/* No projects */}
        {projects.length === 0 && (
          <div className="px-3 pb-2">
            <button
              onClick={openWizard}
              className="w-full px-2 py-1.5 text-xs text-gray-500 hover:text-gray-300 text-center"
            >
              Create a project
            </button>
          </div>
        )}
      </div>

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
