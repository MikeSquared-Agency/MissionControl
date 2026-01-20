import { useStore, useStats } from '../stores/useStore'

interface KingHeaderProps {
  onExit: () => void
}

export function KingHeader({ onExit }: KingHeaderProps) {
  const stats = useStats()
  const clearKingConversation = useStore((s) => s.clearKingConversation)

  return (
    <div className="px-4 py-3 bg-gradient-to-r from-amber-500/10 to-amber-600/5 border-b border-amber-500/20">
      <div className="flex items-center justify-between">
        {/* Left: Crown and title */}
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-full bg-amber-500/20 flex items-center justify-center">
            <span className="text-2xl">👑</span>
          </div>
          <div>
            <h2 className="text-sm font-semibold text-amber-400">King Mode</h2>
            <p className="text-[11px] text-gray-500">AI Orchestrator managing your team</p>
          </div>
        </div>

        {/* Center: Team stats */}
        <div className="flex items-center gap-6 text-[11px]">
          <div className="text-center">
            <p className="text-lg font-mono text-amber-400">{stats.total}</p>
            <p className="text-gray-500">Agents</p>
          </div>
          <div className="text-center">
            <p className="text-lg font-mono text-green-400">{stats.working}</p>
            <p className="text-gray-500">Working</p>
          </div>
          {stats.waiting > 0 && (
            <div className="text-center">
              <p className="text-lg font-mono text-amber-500 animate-pulse">{stats.waiting}</p>
              <p className="text-gray-500">Waiting</p>
            </div>
          )}
          <div className="text-center">
            <p className="text-lg font-mono text-gray-400">${stats.cost.toFixed(2)}</p>
            <p className="text-gray-500">Total Cost</p>
          </div>
          <div className="text-center">
            <p className="text-lg font-mono text-cyan-400">{stats.tokens.toLocaleString()}</p>
            <p className="text-gray-500">Tokens</p>
          </div>
        </div>

        {/* Right: Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={clearKingConversation}
            className="px-3 py-1.5 text-[11px] text-gray-400 hover:text-gray-300 hover:bg-gray-800 rounded transition-colors"
            title="Clear conversation"
          >
            Clear
          </button>
          <button
            onClick={onExit}
            className="px-3 py-1.5 text-[11px] font-medium text-amber-400 bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/30 rounded transition-colors"
            title="Exit King Mode (⌘⇧K)"
          >
            Exit King Mode
          </button>
        </div>
      </div>
    </div>
  )
}
