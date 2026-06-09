import { useState } from "react"
import { cn } from "@/shared/lib/cn"
import { EMPLOYEES } from "@/entities/employee/fixtures"
import { useMiniApp } from "@/shared/miniapp/context"
import type { Meeting } from "@/entities/meeting/types"
import { Avatar, CatCard, Segmented } from "@/shared/ui/cat/primitives"
import { MeetingCard } from "@/components/meetings/meeting-ui"

export function AdminPanel({ meetings }: { meetings: Meeting[] }) {
  const t = useMiniApp().t
  const [tab, setTab] = useState<"meetings" | "users">("meetings")

  return (
    <div className="px-4 pb-7">
      <div className="mb-[18px]">
        <Segmented
          value={tab}
          onChange={setTab}
          options={[
            {
              value: "meetings",
              label: t("allMeetings").split(" ")[0] ?? "Встречи",
            },
            { value: "users", label: t("users") },
          ]}
        />
      </div>
      {tab === "meetings" ? (
        <div>
          <div className="miniapp-section-title mx-1 mb-2.5 font-bold">
            {meetings.length} {t("allMeetings").toLowerCase()}
          </div>
          <div className="flex flex-col gap-[11px]">
            {[...meetings]
              .sort((a, b) =>
                (a.date + a.start).localeCompare(b.date + b.start)
              )
              .map((m) => (
                <MeetingCard key={m.id} m={m} compact />
              ))}
          </div>
        </div>
      ) : (
        <CatCard className="p-1.5">
          {EMPLOYEES.map((e, i) => (
            <div
              key={e.id}
              className={cn(
                "flex items-center gap-[11px] px-2 py-[9px]",
                i < EMPLOYEES.length - 1 && "border-miniapp-border border-b"
              )}
            >
              <Avatar name={e.name} size={38} />
              <div className="min-w-0 flex-1">
                <div className="font-display text-miniapp-text text-[14.5px] font-bold">
                  {e.name}
                </div>
                <div className="text-miniapp-muted truncate text-xs">{e.email}</div>
              </div>
              <span
                className={cn(
                  "font-display rounded-full px-[9px] py-[3px] text-[11px] font-extrabold",
                  e.id === "u1"
                    ? "bg-miniapp-accent-soft text-miniapp-accent"
                    : "bg-miniapp-card-alt text-miniapp-muted"
                )}
              >
                {e.id === "u1" ? `👑 ${t("role_admin")}` : t("role_user")}
              </span>
            </div>
          ))}
        </CatCard>
      )}
    </div>
  )
}
