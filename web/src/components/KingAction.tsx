import type { KingAction as KingActionType } from '../types'

interface KingActionProps {
  action: KingActionType
}

export function KingAction({ action }: KingActionProps) {
  const icons: Record<string, string> = {
    spawn: '🚀',
    kill: '💀',
    message: '💬',
    create_zone: '📁',
    move_agent: '↔️'
  }

  const labels: Record<string, string> = {
    spawn: 'Spawning agent',
    kill: 'Killing agent',
    message: 'Sending message',
    create_zone: 'Creating zone',
    move_agent: 'Moving agent'
  }

  const getDescription = () => {
    switch (action.type) {
      case 'spawn':
        return (
          <span>
            <span className="text-amber-400">{action.agent || 'New agent'}</span>
            {action.persona && (
              <span className="text-gray-500"> as {action.persona}</span>
            )}
            {action.zone && (
              <span className="text-gray-500"> in {action.zone}</span>
            )}
            {action.task && (
              <span className="text-gray-400 block mt-1 text-[11px]">
                "{action.task}"
              </span>
            )}
          </span>
        )
      case 'kill':
        return (
          <span className="text-red-400">{action.agent}</span>
        )
      case 'message':
        return (
          <span>
            <span className="text-blue-400">{action.agent}</span>
            {action.content && (
              <span className="text-gray-400 block mt-1 text-[11px]">
                "{action.content.slice(0, 100)}{action.content.length > 100 ? '...' : ''}"
              </span>
            )}
          </span>
        )
      case 'create_zone':
        return (
          <span className="text-green-400">{action.zone}</span>
        )
      case 'move_agent':
        return (
          <span>
            <span className="text-amber-400">{action.agent}</span>
            <span className="text-gray-500"> → </span>
            <span className="text-green-400">{action.zone}</span>
          </span>
        )
      default:
        return null
    }
  }

  return (
    <div className="flex items-start gap-2 px-3 py-2 bg-amber-500/10 border border-amber-500/20 rounded">
      <span className="text-sm">{icons[action.type] || '⚡'}</span>
      <div className="flex-1 min-w-0">
        <p className="text-[11px] font-medium text-amber-400">
          {labels[action.type] || action.type}
        </p>
        <div className="text-xs text-gray-300">
          {getDescription()}
        </div>
      </div>
    </div>
  )
}
