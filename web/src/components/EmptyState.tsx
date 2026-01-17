interface EmptyStateProps {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  compact?: boolean
}

export function EmptyState({ icon, title, description, action, compact = false }: EmptyStateProps) {
  if (compact) {
    return (
      <div className="px-4 py-6 text-center">
        {icon && <div className="text-2xl mb-2">{icon}</div>}
        <p className="text-sm text-gray-500">{title}</p>
        {description && (
          <p className="text-xs text-gray-600 mt-1">{description}</p>
        )}
        {action && (
          <button
            onClick={action.onClick}
            className="mt-3 text-xs text-blue-500 hover:text-blue-400 transition-colors"
          >
            {action.label}
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="text-center max-w-sm">
        {icon && (
          <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-gray-800/50 flex items-center justify-center text-2xl">
            {icon}
          </div>
        )}
        <h3 className="text-sm font-medium text-gray-300">{title}</h3>
        {description && (
          <p className="mt-2 text-xs text-gray-500">{description}</p>
        )}
        {action && (
          <button
            onClick={action.onClick}
            className="mt-4 px-4 py-2 text-xs font-medium text-white bg-blue-600 hover:bg-blue-500 rounded transition-colors"
          >
            {action.label}
          </button>
        )}
      </div>
    </div>
  )
}

// Pre-built empty states for common scenarios
export function NoAgentsSelected({ onSpawn }: { onSpawn?: () => void }) {
  return (
    <EmptyState
      icon={
        <svg className="w-8 h-8 text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
        </svg>
      }
      title="Select an agent to view details"
      description="Choose an agent from the sidebar, or spawn a new one to get started."
      action={onSpawn ? { label: 'Spawn Agent', onClick: onSpawn } : undefined}
    />
  )
}

export function NoAgentsRunning({ onSpawn }: { onSpawn?: () => void }) {
  return (
    <EmptyState
      icon="🤖"
      title="No agents running"
      description="Spawn an agent to start working on your tasks."
      action={onSpawn ? { label: 'Spawn Agent', onClick: onSpawn } : undefined}
    />
  )
}

export function NoZones({ onCreate }: { onCreate?: () => void }) {
  return (
    <EmptyState
      icon="📁"
      title="No zones created"
      description="Create a zone to organize your agents by project or task."
      action={onCreate ? { label: 'Create Zone', onClick: onCreate } : undefined}
      compact
    />
  )
}

export function NoFindings() {
  return (
    <EmptyState
      icon="🔍"
      title="No findings yet"
      description="The agent hasn't reported any discoveries."
      compact
    />
  )
}

export function NoConversation() {
  return (
    <EmptyState
      icon="💬"
      title="No messages yet"
      description="This agent was just spawned. Messages will appear here."
      compact
    />
  )
}

export function ConnectionLost({ onReconnect }: { onReconnect?: () => void }) {
  return (
    <EmptyState
      icon="📡"
      title="Connection lost"
      description="Unable to connect to the server. Check your connection and try again."
      action={onReconnect ? { label: 'Reconnect', onClick: onReconnect } : undefined}
    />
  )
}
