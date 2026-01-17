interface SkeletonProps {
  className?: string
}

export function Skeleton({ className = '' }: SkeletonProps) {
  return (
    <div
      className={`animate-pulse bg-gray-800 rounded ${className}`}
    />
  )
}

export function SkeletonText({ lines = 1, className = '' }: { lines?: number; className?: string }) {
  return (
    <div className={`space-y-2 ${className}`}>
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton
          key={i}
          className={`h-3 ${i === lines - 1 && lines > 1 ? 'w-3/4' : 'w-full'}`}
        />
      ))}
    </div>
  )
}

export function SkeletonAgentCard() {
  return (
    <div className="mx-2 mb-1 px-2 py-2 rounded bg-gray-800/30">
      <div className="flex items-center gap-2">
        <Skeleton className="w-2 h-2 rounded-full" />
        <Skeleton className="h-3 w-24" />
        <div className="flex-1" />
        <Skeleton className="h-4 w-8 rounded" />
      </div>
      <Skeleton className="h-3 w-full mt-2" />
      <div className="flex items-center gap-2 mt-2">
        <Skeleton className="h-2 w-12" />
        <Skeleton className="h-2 w-10" />
      </div>
    </div>
  )
}

export function SkeletonZoneGroup() {
  return (
    <div className="border-b border-gray-800/50">
      {/* Zone header */}
      <div className="flex items-center gap-2 px-3 py-2">
        <Skeleton className="w-3 h-3" />
        <Skeleton className="w-2 h-2 rounded-full" />
        <Skeleton className="h-3 w-20" />
        <div className="flex-1" />
        <Skeleton className="h-3 w-8" />
      </div>

      {/* Agent cards */}
      <div className="pb-1">
        <SkeletonAgentCard />
        <SkeletonAgentCard />
      </div>
    </div>
  )
}

export function SkeletonSidebar() {
  return (
    <aside className="w-72 flex flex-col bg-gray-900 border-r border-gray-800/50">
      <div className="flex-1 overflow-hidden">
        <SkeletonZoneGroup />
        <SkeletonZoneGroup />
      </div>
      <div className="px-3 py-2 border-t border-gray-800/50">
        <Skeleton className="h-3 w-32" />
      </div>
    </aside>
  )
}

export function SkeletonConversation() {
  return (
    <div className="flex-1 overflow-hidden px-4 py-4 space-y-4">
      {/* Assistant message */}
      <div className="flex justify-start">
        <div className="max-w-[70%] bg-gray-800/50 rounded-lg p-4">
          <SkeletonText lines={3} />
        </div>
      </div>

      {/* User message */}
      <div className="flex justify-end">
        <div className="max-w-[60%] bg-blue-600/20 rounded-lg p-4">
          <SkeletonText lines={2} />
        </div>
      </div>

      {/* Assistant message */}
      <div className="flex justify-start">
        <div className="max-w-[70%] bg-gray-800/50 rounded-lg p-4">
          <SkeletonText lines={4} />
        </div>
      </div>
    </div>
  )
}
