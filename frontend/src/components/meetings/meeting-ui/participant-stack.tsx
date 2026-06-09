import { cn } from "@/shared/lib/cn"
import { stackBadgeVars } from "@/shared/tma/surface-vars"
import { emailsToPeople } from "@/entities/employee/fixtures"
import { Avatar } from "@/shared/ui/cat/primitives"

export function ParticipantStack({
  emails,
  max = 4,
  size = 28,
}: {
  emails: string[]
  max?: number
  size?: number
}) {
  const people = emailsToPeople(emails)
  const shown = people.slice(0, max)
  const extra = people.length - shown.length
  return (
    <div className="flex items-center">
      {shown.map((per, i) => (
        <div key={i} className={cn(i > 0 && "-ml-2.5")}>
          <Avatar name={per.name} size={size} ring />
        </div>
      ))}
      {extra > 0 && (
        <div
          className={cn(
            "tma-stack-badge -ml-2.5 flex items-center justify-center rounded-full",
            "bg-tma-card-alt font-display text-tma-muted font-extrabold",
            "shadow-[0_0_0_2px_var(--tma-bg)]"
          )}
          style={stackBadgeVars(size)}
        >
          +{extra}
        </div>
      )}
    </div>
  )
}
