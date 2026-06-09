import { useMemo } from "react"
import { Link, useNavigate } from "@tanstack/react-router"
import { cn } from "@/shared/lib/cn"
import { MINIAPP_NOW } from "@/entities/meeting/constants"
import { useMiniAppAuth } from "@/shared/auth/auth-context"
import { useMiniApp } from "@/shared/miniapp/context"
import { splitHomeMeetings } from "@/entities/meeting/lib/home-meetings"
import { useMyMeetings } from "@/entities/meeting/queries"
import { hueSurfaceVars } from "@/shared/miniapp/surface-vars"
import { CatIcon } from "@/shared/ui/cat/primitives"
import { Paw } from "@/shared/ui/cat/paw"
import {
  EmptyState,
  MeetingCard,
  meetCount,
  SectionTitle,
} from "@/components/meetings/meeting-ui"

export function HomePage() {
  const p = useMiniApp()
  const { user } = useMiniAppAuth()
  const navigate = useNavigate()
  // scope=upcoming: past-today meetings are omitted (backend window starts at now).
  const { data: meetings = [] } = useMyMeetings("upcoming")
  const t = p.t
  const { todayMeetings: today, upcomingMeetings: upcoming } = useMemo(
    () => splitHomeMeetings(meetings, MINIAPP_NOW),
    [meetings]
  )
  const firstName = (user?.name ?? "").split(" ")[0]

  const actions = [
    {
      key: "create",
      icon: "calendar" as const,
      label: t("create"),
      hue: 45,
      to: "/meetings/create" as const,
    },
    {
      key: "find",
      icon: "search" as const,
      label: t("findTime"),
      hue: 180,
      to: "/checker" as const,
    },
    {
      key: "sched",
      icon: "users" as const,
      label: t("schedule"),
      hue: 300,
      to: "/profile/colleague" as const,
    },
  ]

  const openMeeting = (id: string) => {
    void navigate({
      to: "/meetings/$meetingId",
      params: { meetingId: id },
      search: { scope: "upcoming" },
    })
  }

  return (
    <div className="flex flex-col gap-[22px] px-4 pb-7 pt-4">
      <div className="bg-miniapp-hero border-miniapp-accent-line relative overflow-hidden rounded-3xl border px-5 pb-[22px] pt-5">
        <Paw
          size={150}
          className="absolute -bottom-[34px] -right-7 -rotate-[18deg]"
        />
        <div className="relative">
          <div
            className={cn(
              "mb-1 text-[13.5px] font-bold",
              p.dark ? "text-miniapp-accent/95" : "text-miniapp-accent"
            )}
          >
            {t("greet_morning")} 🐾
          </div>
          <div className="miniapp-heading mb-2.5 text-[28px] leading-[1.05]">
            {firstName}!
          </div>
          <div className="text-miniapp-text text-[14.5px] font-semibold opacity-85">
            {today.length > 0
              ? `${t("today")}: ${meetCount(today.length, p.lang)}`
              : t("nothingToday")}
          </div>
        </div>
      </div>

      <div>
        <SectionTitle>{t("quick")}</SectionTitle>
        <div className="grid grid-cols-2 gap-[11px]">
          {actions.map((a) => (
            <Link
              key={a.key}
              to={a.to}
              className={cn(
                "flex cursor-pointer flex-col gap-[11px] rounded-[18px]",
                "border-miniapp-border bg-miniapp-card border px-3.5 py-[15px]",
                "shadow-miniapp-sm text-left no-underline transition-transform duration-150"
              )}
            >
              <div
                className="miniapp-hue-surface flex size-[42px] items-center justify-center rounded-[13px]"
                style={hueSurfaceVars(a.hue, p.dark)}
              >
                <CatIcon
                  name={a.icon}
                  size={23}
                  className="text-miniapp-hue-fg"
                  sw={2.1}
                />
              </div>
              <span className="font-display text-miniapp-text text-[14.5px] font-bold leading-[1.15]">
                {a.label}
              </span>
            </Link>
          ))}
        </div>
      </div>

      <div>
        <SectionTitle
          action={today.length ? t("seeAll") : null}
          onAction={() =>
            void navigate({ to: "/meetings", search: { scope: "upcoming" } })
          }
        >
          {t("today")}
        </SectionTitle>
        {today.length === 0 ? (
          <div className="border-miniapp-border bg-miniapp-card overflow-hidden rounded-[20px] border">
            <EmptyState emoji="😴" title={t("nothingToday")} />
          </div>
        ) : (
          <div className="flex flex-col gap-[11px]">
            {today.map((m) => (
              <MeetingCard key={m.id} m={m} onClick={() => openMeeting(m.id)} />
            ))}
          </div>
        )}
      </div>

      {upcoming.length > 0 && (
        <div>
          <SectionTitle
            action={t("seeAll")}
            onAction={() =>
              void navigate({ to: "/meetings", search: { scope: "upcoming" } })
            }
          >
            {t("upcoming")}
          </SectionTitle>
          <div className="flex flex-col gap-[11px]">
            {upcoming.map((m) => (
              <MeetingCard key={m.id} m={m} onClick={() => openMeeting(m.id)} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
