import { useStore, useStats, useConnectionStatus } from '../stores/useStore'
import type { ConnectionStatus } from '../types'

interface HeaderProps {
  onSpawnClick: () => void
  onNewZoneClick: () => void
  onSettingsClick: () => void
}

export function Header({ onSpawnClick, onNewZoneClick, onSettingsClick }: HeaderProps) {
  const connectionStatus = useConnectionStatus()
  const stats = useStats()
  const kingMode = useStore((s) => s.kingMode)
  const setKingMode = useStore((s) => s.setKingMode)

  const statusConfig: Record<ConnectionStatus, { color: string; text: string; pulse?: boolean }> = {
    connecting: { color: 'bg-yellow-500', text: 'Connecting...', pulse: true },
    connected: { color: 'bg-green-500', text: 'Connected' },
    disconnected: { color: 'bg-gray-500', text: 'Disconnected' }
  }

  const { color, text, pulse } = statusConfig[connectionStatus]

  return (
    <header className="h-12 flex items-center justify-between px-4 bg-gray-900 border-b border-gray-800/50">
      {/* Left: Logo and connection status */}
      <div className="flex items-center gap-4">
        <h1 className="text-sm font-semibold text-gray-100 tracking-tight">
          MissionControl
          <span className="ml-1.5 text-[10px] font-normal text-gray-600">v3</span>
        </h1>

        {/* Connection status */}
        <button
          onClick={() => {
            if (connectionStatus === 'disconnected') {
              // Trigger reconnect through WebSocket hook
              window.location.reload()
            }
          }}
          className="flex items-center gap-1.5 text-[11px] text-gray-500 hover:text-gray-400 transition-colors"
          title={connectionStatus === 'disconnected' ? 'Click to reconnect' : undefined}
        >
          <span className={`w-1.5 h-1.5 rounded-full ${color} ${pulse ? 'animate-pulse' : ''}`} />
          <span>{text}</span>
        </button>
      </div>

      {/* Center: Stats bar */}
      <div className="flex items-center gap-4 text-[11px]">
        {/* Total agents */}
        <div className="text-gray-500">
          <span className="font-mono text-gray-400">{stats.total}</span> agents
        </div>

        {/* Total tokens */}
        <div className="text-gray-500">
          <span className="font-mono text-gray-400">{formatTokens(stats.tokens)}</span> tokens
        </div>

        {/* Total cost */}
        <div className="text-gray-500">
          <span className="font-mono text-gray-400">${stats.cost.toFixed(2)}</span>
        </div>

        {/* Working count */}
        {stats.working > 0 && (
          <div className="flex items-center gap-1 text-green-500">
            <span className="w-1.5 h-1.5 rounded-full bg-green-500" />
            <span>{stats.working} working</span>
          </div>
        )}

        {/* Waiting count */}
        {stats.waiting > 0 && (
          <div className="flex items-center gap-1 text-amber-500 animate-pulse">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
            <span>{stats.waiting} waiting</span>
          </div>
        )}

        {/* Error count */}
        {stats.error > 0 && (
          <div className="flex items-center gap-1 text-red-500">
            <span className="w-1.5 h-1.5 rounded-full bg-red-500" />
            <span>{stats.error} error</span>
          </div>
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-2">
        {/* King Mode toggle */}
        <button
          onClick={() => setKingMode(!kingMode)}
          className={`
            flex items-center gap-1.5 px-2 py-1 text-[11px] font-medium rounded transition-colors
            ${kingMode
              ? 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
              : 'text-gray-500 hover:text-gray-400 hover:bg-gray-800'
            }
          `}
          title="Toggle King Mode (⌘⇧K)"
        >
          <span>👑</span>
          <span>King</span>
        </button>

        {/* Spawn agent button (hidden in King mode) */}
        {!kingMode && (
          <button
            onClick={onSpawnClick}
            className="px-2 py-1 text-[11px] font-medium text-gray-300 bg-blue-600 hover:bg-blue-500 rounded transition-colors"
            title="Spawn Agent (⌘N)"
          >
            + Agent
          </button>
        )}

        {/* New zone button (hidden in King mode) */}
        {!kingMode && (
          <button
            onClick={onNewZoneClick}
            className="px-2 py-1 text-[11px] font-medium text-gray-400 hover:text-gray-300 hover:bg-gray-800 rounded transition-colors"
            title="New Zone (⌘⇧N)"
          >
            + Zone
          </button>
        )}

        {/* Settings button */}
        <button
          onClick={onSettingsClick}
          className="p-1.5 text-gray-500 hover:text-gray-400 hover:bg-gray-800 rounded transition-colors"
          title="Settings (⌘,)"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z"
            />
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </button>
      </div>
    </header>
  )
}

function formatTokens(tokens: number): string {
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(1)}M`
  }
  if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}k`
  }
  return String(tokens)
}
