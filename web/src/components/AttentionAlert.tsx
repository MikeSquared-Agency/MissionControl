import type { AttentionRequest } from '../types'

interface AttentionAlertProps {
  attention: AttentionRequest
  onRespond?: (response: string) => void
}

export function AttentionAlert({ attention, onRespond }: AttentionAlertProps) {
  const { type, message, retryable, retryIn } = attention

  const icons: Record<string, string> = {
    question: '❓',
    permission: '🔐',
    error: '❌',
    complete: '✅'
  }

  const titles: Record<string, string> = {
    question: 'Agent has a question',
    permission: 'Permission required',
    error: 'Error occurred',
    complete: 'Task complete'
  }

  const bgColors: Record<string, string> = {
    question: 'bg-amber-500/10 border-amber-500/20',
    permission: 'bg-amber-500/10 border-amber-500/20',
    error: 'bg-red-500/10 border-red-500/20',
    complete: 'bg-green-500/10 border-green-500/20'
  }

  const textColors: Record<string, string> = {
    question: 'text-amber-400',
    permission: 'text-amber-400',
    error: 'text-red-400',
    complete: 'text-green-400'
  }

  return (
    <div className={`px-4 py-3 border-b ${bgColors[type]}`}>
      <div className="flex items-start gap-3">
        {/* Icon */}
        <span className="text-lg mt-0.5">{icons[type]}</span>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <p className={`text-xs font-medium ${textColors[type]}`}>
            {titles[type]}
          </p>
          <p className="text-xs text-gray-300 mt-0.5">
            {message}
          </p>

          {/* Retry info for errors */}
          {type === 'error' && retryIn && (
            <p className="text-[10px] text-gray-500 mt-1">
              {retryable ? `Will retry in ${retryIn}s` : 'Cannot retry automatically'}
            </p>
          )}

          {/* Quick actions */}
          {(type === 'question' || type === 'permission') && onRespond && (
            <div className="mt-2 flex items-center gap-2">
              {type === 'permission' ? (
                <>
                  <button
                    onClick={() => onRespond('Allow')}
                    className="px-2.5 py-1 text-[11px] font-medium text-green-400 bg-green-500/10 hover:bg-green-500/20 border border-green-500/30 rounded transition-colors"
                  >
                    Allow
                  </button>
                  <button
                    onClick={() => onRespond('Deny')}
                    className="px-2.5 py-1 text-[11px] font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 rounded transition-colors"
                  >
                    Deny
                  </button>
                </>
              ) : (
                <>
                  <button
                    onClick={() => onRespond('Yes')}
                    className="px-2.5 py-1 text-[11px] font-medium text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded transition-colors"
                  >
                    Yes
                  </button>
                  <button
                    onClick={() => onRespond('No')}
                    className="px-2.5 py-1 text-[11px] font-medium text-gray-400 bg-gray-500/10 hover:bg-gray-500/20 border border-gray-500/30 rounded transition-colors"
                  >
                    No
                  </button>
                </>
              )}
            </div>
          )}

          {/* Retry button for errors */}
          {type === 'error' && retryable && onRespond && (
            <button
              onClick={() => onRespond('retry')}
              className="mt-2 px-2.5 py-1 text-[11px] font-medium text-amber-400 bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/30 rounded transition-colors"
            >
              Retry now
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
