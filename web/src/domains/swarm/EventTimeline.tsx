import { useSwarmEvents } from '../../stores/useSwarmStore'
import type { WarrenSSEEvent } from '../../types/swarm'

export function EventTimeline() {
  const events = useSwarmEvents()

  return (
    <div className="rounded-lg bg-gray-800/50 border border-gray-800 p-3">
      <h3 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-3">
        Live Events
        {events.length > 0 && (
          <span className="ml-2 text-gray-600">({events.length})</span>
        )}
      </h3>

      {events.length === 0 ? (
        <div className="text-xs text-gray-600 text-center py-4">
          Waiting for events from Warren...
        </div>
      ) : (
        <div className="space-y-1 max-h-48 overflow-y-auto">
          {events.map((event, i) => (
            <EventRow key={`${event.timestamp}-${i}`} event={event} />
          ))}
        </div>
      )}
    </div>
  )
}

function EventRow({ event }: { event: WarrenSSEEvent }) {
  const time = new Date(event.timestamp).toLocaleTimeString()

  return (
    <div className="flex items-center gap-2 py-1 px-2 rounded hover:bg-gray-900/50 transition-colors">
      {/* Timestamp */}
      <span className="text-[10px] text-gray-600 font-mono w-16 flex-shrink-0">
        {time}
      </span>

      {/* Event type badge */}
      <span className={`
        px-1.5 py-0.5 text-[10px] font-medium rounded flex-shrink-0
        ${getEventBadgeStyle(event.type)}
      `}>
        {event.type}
      </span>

      {/* Agent name */}
      {event.agent && (
        <span className="text-[10px] text-gray-400 truncate flex-shrink-0">
          {event.agent}
        </span>
      )}

      {/* Event summary */}
      <span className="text-[10px] text-gray-600 truncate flex-1">
        {formatEventData(event)}
      </span>
    </div>
  )
}

function getEventBadgeStyle(type: string): string {
  if (type.includes('error') || type.includes('fail')) {
    return 'bg-red-500/20 text-red-400'
  }
  if (type.includes('warn') || type.includes('alert')) {
    return 'bg-yellow-500/20 text-yellow-400'
  }
  if (type.includes('spawn') || type.includes('start') || type.includes('connect')) {
    return 'bg-green-500/20 text-green-400'
  }
  if (type.includes('stop') || type.includes('kill') || type.includes('disconnect')) {
    return 'bg-orange-500/20 text-orange-400'
  }
  return 'bg-gray-700 text-gray-400'
}

function formatEventData(event: WarrenSSEEvent): string {
  if (!event.data) return ''
  if (typeof event.data === 'string') return event.data
  try {
    const str = JSON.stringify(event.data)
    return str.length > 80 ? str.slice(0, 80) + '...' : str
  } catch {
    return ''
  }
}
