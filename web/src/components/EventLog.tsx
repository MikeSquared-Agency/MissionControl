import { useAgentStore } from '../stores/agentStore'
import type { Event } from '../types'

export function EventLog() {
  const events = useAgentStore((s) => s.events)
  const selectedAgentId = useAgentStore((s) => s.selectedAgentId)

  // Filter events for selected agent, or show all if none selected
  const filteredEvents = selectedAgentId
    ? events.filter((e) => e.agent_id === selectedAgentId)
    : events

  if (filteredEvents.length === 0) {
    return (
      <div className="p-6 text-center text-gray-500">
        <p>No events yet.</p>
        <p className="text-sm mt-2">Events will appear here as agents work.</p>
      </div>
    )
  }

  return (
    <div className="p-4 space-y-2 font-mono text-sm overflow-auto max-h-full">
      {filteredEvents.map((event, i) => (
        <EventItem key={i} event={event} />
      ))}
    </div>
  )
}

function EventItem({ event }: { event: Event }) {
  const typeColors: Record<string, string> = {
    turn: 'text-blue-400',
    thinking: 'text-gray-300',
    tool_call: 'text-yellow-400',
    tool_result: 'text-green-400',
    output: 'text-gray-300',
    error: 'text-red-400',
    agent_error: 'text-red-400'
  }

  const color = typeColors[event.type] || 'text-gray-400'

  return (
    <div className={`${color} break-words`}>
      <span className="text-gray-600">[{event.agent_id?.slice(0, 6) || '???'}]</span>{' '}
      {event.type === 'turn' && (
        <span>Turn {event.turn}</span>
      )}
      {event.type === 'thinking' && (
        <span>{event.content}</span>
      )}
      {event.type === 'tool_call' && (
        <span>
          $ {event.tool}{' '}
          {event.args && (
            <span className="text-gray-500">
              {JSON.stringify(event.args).slice(0, 100)}
            </span>
          )}
        </span>
      )}
      {event.type === 'tool_result' && (
        <span className="text-gray-400">
          {event.result?.slice(0, 200)}
          {event.result && event.result.length > 200 ? '...' : ''}
        </span>
      )}
      {event.type === 'output' && (
        <span>{event.content}</span>
      )}
      {(event.type === 'error' || event.type === 'agent_error') && (
        <span>{event.error || event.content}</span>
      )}
    </div>
  )
}
