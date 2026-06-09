import { useState } from "react"
import { useEmployeeSearch } from "@/entities/employee/queries"
import { useColleagueSchedule } from "@/entities/meeting/queries"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar, CatCard, CatIcon } from "@/shared/ui/cat/primitives"
import { EmptyState, MeetingCard } from "@/components/meetings/meeting-ui"

export function ColleagueSchedule() {
  const t = useTmaApp().t
  const [picked, setPicked] = useState<{
    id: string
    name: string
    email: string
    dept?: string
  } | null>(null)
  const [search, setSearch] = useState("")

  const { data: searchResults = [] } = useEmployeeSearch(search)
  const matches = searchResults.slice(0, 6)

  const { data: colleagueMeetings = [], isLoading: loadingSchedule } =
    useColleagueSchedule(picked?.email ?? "", "all")

  if (picked) {
    const list = [...colleagueMeetings].sort((a, b) =>
      (a.date + a.start).localeCompare(b.date + b.start)
    )
    return (
      <div className="px-4 pb-7">
        <button
          type="button"
          onClick={() => setPicked(null)}
          className="font-display text-tma-accent mb-3.5 flex cursor-pointer items-center gap-1 border-none bg-transparent text-sm font-bold"
        >
          <CatIcon name="chevL" size={16} sw={2.4} /> {t("searchColleague")}
        </button>
        <div className="mb-2 flex items-center gap-3">
          <Avatar name={picked.name} size={52} />
          <div>
            <div className="font-display text-tma-text text-[19px] font-extrabold">
              {picked.name}
            </div>
            <div className="text-tma-muted text-[13px]">{picked.dept}</div>
          </div>
        </div>
        <div className="bg-tma-card-alt text-tma-muted mb-[18px] inline-flex items-center gap-1.5 rounded-full px-[11px] py-[5px] text-xs font-bold">
          👁️ {t("viewOnly")}
        </div>
        {loadingSchedule ? (
          <div className="text-tma-muted p-5 text-center text-sm">
            {t("loading")}
          </div>
        ) : list.length === 0 ? (
          <div className="border-tma-border bg-tma-card overflow-hidden rounded-[20px] border">
            <EmptyState
              emoji="😺"
              title="Свободен"
              sub="Нет встреч в расписании"
            />
          </div>
        ) : (
          <div className="flex flex-col gap-[11px]">
            {list.map((m) => (
              <MeetingCard key={m.id} m={m} compact />
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="px-4 pb-7">
      <h2 className="tma-heading mb-1.5 text-[23px]">{t("searchColleague")}</h2>
      <p className="text-tma-muted mb-[18px] text-sm">
        Посмотри расписание любого сотрудника
      </p>
      <div className="relative">
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t("searchPeople")}
          className="tma-input pl-[42px]"
          autoFocus
        />
        <span className="text-tma-faint pointer-events-none absolute left-[13px] top-[15px]">
          <CatIcon name="search" size={19} sw={2} />
        </span>
      </div>
      {search === "" ? (
        <div className="text-tma-faint mt-4 p-5 text-center text-sm">
          {t("typeToSearch")} 🐾
        </div>
      ) : (
        <div className="mt-3 flex flex-col gap-2">
          {matches.map((e) => (
            <CatCard
              key={e.id}
              interactive
              onClick={() => {
                setPicked(e)
                setSearch("")
              }}
              className="flex items-center gap-[11px] p-3"
            >
              <Avatar name={e.name} size={40} />
              <div className="min-w-0 flex-1">
                <div className="font-display text-tma-text text-[15px] font-bold">
                  {e.name}
                </div>
                <div className="text-tma-muted text-[12.5px]">
                  {e.dept} · {e.email}
                </div>
              </div>
              <span className="text-tma-faint">
                <CatIcon name="chevR" size={18} sw={2.2} />
              </span>
            </CatCard>
          ))}
        </div>
      )}
    </div>
  )
}
