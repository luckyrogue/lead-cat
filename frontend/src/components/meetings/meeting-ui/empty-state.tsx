export function EmptyState({
  emoji = "🐈",
  title,
  sub,
}: {
  emoji?: string
  title: string
  sub?: string
}) {
  return (
    <div className="px-6 py-11 text-center">
      <div className="animate-lc-bob mb-3 text-[52px]">{emoji}</div>
      <div className="tma-heading mb-1.5 text-[17px]">{title}</div>
      {sub && (
        <div className="mx-auto max-w-[240px] text-sm leading-snug text-tma-muted">
          {sub}
        </div>
      )}
    </div>
  )
}
