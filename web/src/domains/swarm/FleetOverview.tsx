import { useFleetSummary, useSwarmStore } from '../../stores/useSwarmStore'

export function FleetOverview() {
  const summary = useFleetSummary()
  const errorCount = useSwarmStore((s) => Object.keys(s.overview?.errors ?? {}).length)

  if (!summary) return null

  const cards = [
    {
      label: 'Agents',
      value: summary.totalAgents,
      sub: `${summary.readyCount} ready / ${summary.sleepingCount} sleeping`,
      color: summary.totalAgents > 0 ? 'text-green-400' : 'text-gray-500'
    },
    {
      label: 'Active Tasks',
      value: summary.activeTasks,
      sub: `${summary.dlqDepth} in DLQ`,
      color: summary.activeTasks > 0 ? 'text-blue-400' : 'text-gray-500'
    },
    {
      label: 'Prompts',
      value: summary.promptCount,
      sub: `${summary.collectionCount} collections`,
      color: 'text-purple-400'
    },
    {
      label: 'Services',
      value: `${5 - errorCount}/5`,
      sub: errorCount > 0 ? `${errorCount} degraded` : 'All healthy',
      color: errorCount === 0 ? 'text-green-400' : errorCount >= 3 ? 'text-red-400' : 'text-yellow-400'
    }
  ]

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {cards.map((card) => (
        <div
          key={card.label}
          className="p-3 rounded-lg bg-gray-800/50 border border-gray-800"
        >
          <div className="text-[10px] uppercase tracking-wider text-gray-500 mb-1">
            {card.label}
          </div>
          <div className={`text-2xl font-bold ${card.color}`}>
            {card.value}
          </div>
          <div className="text-[10px] text-gray-600 mt-0.5">
            {card.sub}
          </div>
        </div>
      ))}
    </div>
  )
}
