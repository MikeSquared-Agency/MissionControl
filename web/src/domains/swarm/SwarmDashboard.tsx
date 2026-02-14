import { useState } from 'react'
import { useSwarmPolling } from '../../hooks/useSwarmPolling'
import { useWarrenSSE } from '../../hooks/useWarrenSSE'
import { useSwarmLoading, useSwarmStore } from '../../stores/useSwarmStore'
import { FleetOverview } from './FleetOverview'
import { ServiceStatus } from './ServiceStatus'
import { TaskPipeline } from './TaskPipeline'
import { AgentGrid } from './AgentGrid'
import { EventTimeline } from './EventTimeline'

type SubTab = 'live' | 'schedule'

export function SwarmDashboard() {
  const [subTab, setSubTab] = useState<SubTab>('live')
  const loading = useSwarmLoading()
  const lastFetched = useSwarmStore((s) => s.lastFetched)

  // Poll when on the live tab
  useSwarmPolling({ enabled: subTab === 'live' })

  // SSE always active when mounted
  useWarrenSSE()

  return (
    <div className="h-full flex flex-col bg-gray-900">
      {/* Sub-tab bar */}
      <div className="flex items-center gap-1 px-3 py-2 border-b border-gray-800">
        {(['live', 'schedule'] as SubTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setSubTab(tab)}
            className={`
              px-3 py-1.5 text-xs font-medium rounded transition-colors
              ${subTab === tab
                ? 'bg-purple-600 text-white'
                : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800'
              }
            `}
          >
            {tab === 'live' ? 'Live' : 'Schedule'}
          </button>
        ))}
        {lastFetched && (
          <span className="ml-auto text-[10px] text-gray-600">
            Updated {new Date(lastFetched).toLocaleTimeString()}
          </span>
        )}
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-auto">
        {subTab === 'live' ? (
          loading && !lastFetched ? (
            <LoadingSkeleton />
          ) : (
            <LiveView />
          )
        ) : (
          <SchedulePlaceholder />
        )}
      </div>
    </div>
  )
}

function LiveView() {
  return (
    <div className="p-4 space-y-4">
      {/* Row 1: Fleet overview (full width) */}
      <FleetOverview />

      {/* Row 2: 3-column grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <ServiceStatus />
        <TaskPipeline />
        <AgentGrid />
      </div>

      {/* Row 3: Event timeline (full width) */}
      <EventTimeline />
    </div>
  )
}

function SchedulePlaceholder() {
  return (
    <div className="flex items-center justify-center h-full">
      <div className="text-center">
        <div className="text-gray-600 text-sm">Schedule View</div>
        <div className="text-gray-700 text-xs mt-1">Coming in Phase 2</div>
      </div>
    </div>
  )
}

function LoadingSkeleton() {
  return (
    <div className="p-4 space-y-4 animate-pulse">
      {/* Fleet overview skeleton */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="h-20 bg-gray-800/50 rounded-lg" />
        ))}
      </div>
      {/* 3 column skeleton */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-48 bg-gray-800/50 rounded-lg" />
        ))}
      </div>
      {/* Timeline skeleton */}
      <div className="h-40 bg-gray-800/50 rounded-lg" />
    </div>
  )
}
