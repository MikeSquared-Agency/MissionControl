interface FindingsProps {
  findings: string[]
}

export function Findings({ findings }: FindingsProps) {
  if (findings.length === 0) {
    return null
  }

  return (
    <div className="mt-6 pt-4 border-t border-gray-800">
      <h3 className="text-[10px] uppercase tracking-wider text-gray-600 font-medium mb-2">
        Findings
      </h3>
      <ul className="space-y-1.5">
        {findings.map((finding, i) => (
          <li key={i} className="flex items-start gap-2 text-xs">
            <span className="text-green-500 mt-0.5 flex-shrink-0">•</span>
            <span className="text-gray-400">{finding}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
