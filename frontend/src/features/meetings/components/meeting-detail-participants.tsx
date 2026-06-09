import { cn } from "@/shared/lib/cn"
import { useTmaApp } from "@/shared/tma/context"
import { emailsToPeople } from "@/entities/employee/fixtures"
import { partWord } from "@/entities/meeting/lib/format"
import type { Meeting } from "@/entities/meeting/types"
import { Avatar, CatCard } from "@/shared/ui/cat/primitives"

export function MeetingDetailParticipants({ m }: { m: Meeting }) {
  const t = useTmaApp().t
  const allPeople = emailsToPeople([m.organizer, ...m.participants])

  return (
    <div className="mt-[18px]">
      <div className="mb-2.5 flex items-center justify-between">
        <span className="tma-heading text-base">{t("addPeople")}</span>
        <span className="text-tma-muted text-[13px] font-bold">
          {allPeople.length} {partWord(allPeople.length, t)}
        </span>
      </div>
      <CatCard className="p-1.5">
        {allPeople.map((per, i) => (
          <div
            key={i}
            className={cn(
              "flex items-center gap-[11px] p-2",
              i < allPeople.length - 1 && "border-tma-border border-b"
            )}
          >
            <Avatar name={per.name} size={36} />
            <div className="min-w-0 flex-1">
              <div className="font-display text-tma-text truncate text-[14.5px] font-bold">
                {per.name}
              </div>
              <div className="text-tma-muted truncate text-xs">{per.email}</div>
            </div>
            {per.email === m.organizer ? (
              <span className="bg-tma-accent-soft text-tma-accent rounded-full px-2 py-[3px] text-[11px] font-extrabold">
                👑
              </span>
            ) : per.tg ? (
              <span className="text-[15px]" title="Telegram">
                ✈️
              </span>
            ) : (
              <span className="text-tma-faint text-[11px] font-bold">
                email
              </span>
            )}
          </div>
        ))}
      </CatCard>
    </div>
  )
}
