import { usePipelineSummary } from '../../stores/useSwarmStore'

export function TaskPipeline() {
  const pipeline = usePipelineSummary()

  if (!pipeline) {
    return (
      <div className="rounded-lg bg-gray-800/50 border border-gray-800 p-3">
        <h3 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-3">
          Task Pipeline
        </h3>
        <div className="text-xs text-gray-600 text-center py-6">
          No dispatch data available
        </div>
      </div>
    )
  }

  const stages = [
    { label: 'Pending', value: pipeline.pending, color: 'bg-gray-500' },
    { label: 'In Progress', value: pipeline.inProgress, color: 'bg-blue-500' },
    { label: 'Completed', value: pipeline.completed, color: 'bg-green-500' },
    { label: 'Failed', value: pipeline.failed, color: 'bg-red-500' }
  ]

  const total = pipeline.total || 1 // avoid division by zero

  return (
    <div className="rounded-lg bg-gray-800/50 border border-gray-800 p-3">
      <h3 className="text-xs font-medium text-gray-400 uppercase tracking-wider mb-3">
        Task Pipeline
      </h3>

      {/* Progress bar */}
      <div className="flex h-2 rounded-full overflow-hidden bg-gray-900 mb-3">
        {stages.map((stage) => {
          const width = (stage.value / total) * 100
          if (width === 0) return null
          return (
            <div
              key={stage.label}
              className={`${stage.color} transition-all duration-500`}
              style={{ width: `${width}%` }}
              title={`${stage.label}: ${stage.value}`}
            />
          )
        })}
      </div>

      {/* Stage counts */}
      <div className="grid grid-cols-2 gap-2">
        {stages.map((stage) => (
          <div key={stage.label} className="flex items-center gap-2">
            <span className={`w-2 h-2 rounded-full ${stage.color}`} />
            <span className="text-[10px] text-gray-500">{stage.label}</span>
            <span className="text-xs font-medium text-gray-300 ml-auto">
              {stage.value}
            </span>
          </div>
        ))}
      </div>

      {/* DLQ */}
      {pipeline.dlqDepth > 0 && (
        <div className="mt-3 p-2 rounded bg-red-500/10 border border-red-500/20">
          <div className="flex items-center justify-between">
            <span className="text-[10px] text-red-400">Dead Letter Queue</span>
            <span className="text-xs font-bold text-red-400">{pipeline.dlqDepth}</span>
          </div>
        </div>
      )}
    </div>
  )
}
