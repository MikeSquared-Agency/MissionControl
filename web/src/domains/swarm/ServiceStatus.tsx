import { useSwarmOverview } from '../../stores/useSwarmStore'

interface ServiceInfo {
  name: string
  key: string
  metric?: string
}

const SERVICES: ServiceInfo[] = [
  { name: 'Warren', key: 'warren', metric: 'agents connected' },
  { name: 'Chronicle', key: 'chronicle', metric: 'events tracked' },
  { name: 'Dispatch', key: 'dispatch', metric: 'tasks managed' },
  { name: 'PromptForge', key: 'promptforge', metric: 'prompts' },
  { name: 'Alexandria', key: 'alexandria', metric: 'collections' }
]

export function ServiceStatus() {
  const overview = useSwarmOverview()

  if (!overview) return null

  return (
    <div className="rounded-lg bg-gray-800/50 border border-gray-800 p-3">
      <h3 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-3">
        Service Health
      </h3>
      <div className="space-y-2">
        {SERVICES.map((svc) => {
          const isDown = !!overview.errors[svc.key]
          const errorMsg = overview.errors[svc.key]
          const keyMetric = getKeyMetric(overview, svc.key)

          return (
            <div
              key={svc.key}
              className="flex items-center gap-2 p-2 rounded bg-gray-900/50"
            >
              {/* Status dot */}
              <span
                className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  isDown ? 'bg-red-500' : 'bg-green-500'
                }`}
              />

              {/* Name */}
              <span className="text-xs font-medium text-gray-300 flex-1">
                {svc.name}
              </span>

              {/* Metric or error */}
              <span className={`text-[10px] ${isDown ? 'text-red-400' : 'text-gray-500'}`}>
                {isDown ? truncate(errorMsg, 30) : keyMetric}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function getKeyMetric(overview: NonNullable<ReturnType<typeof useSwarmOverview>>, key: string): string {
  switch (key) {
    case 'warren':
      return overview.warren?.agents
        ? `${overview.warren.agents.length} agents`
        : 'connected'
    case 'chronicle': {
      const depth = overview.chronicle?.dlq?.depth
      return depth !== undefined ? `DLQ: ${depth}` : 'ok'
    }
    case 'dispatch': {
      const active = overview.dispatch?.stats?.in_progress
      return active !== undefined ? `${active} active` : 'ok'
    }
    case 'promptforge':
      return overview.promptforge?.prompt_count !== undefined
        ? `${overview.promptforge.prompt_count} prompts`
        : 'ok'
    case 'alexandria':
      return overview.alexandria?.collection_count !== undefined
        ? `${overview.alexandria.collection_count} collections`
        : 'ok'
    default:
      return 'ok'
  }
}

function truncate(str: string | undefined, max: number): string {
  if (!str) return 'down'
  return str.length > max ? str.slice(0, max) + '...' : str
}
