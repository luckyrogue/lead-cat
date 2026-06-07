import { cn } from "@/shared/lib/cn"

export function MeetingTitlePreview({
  label,
  title,
  className,
}: {
  label: string
  title: string
  className?: string
}) {
  return (
    <div
      className={cn(
        "mb-4 rounded-2xl border border-dashed border-tma-border-strong",
        "bg-tma-card-alt px-[15px] py-[13px]",
        className
      )}
    >
      <div className="tma-label mb-[5px] tracking-wide">{label}</div>
      <div className="tma-heading break-words text-[17px] leading-snug">
        {title}
      </div>
    </div>
  )
}
