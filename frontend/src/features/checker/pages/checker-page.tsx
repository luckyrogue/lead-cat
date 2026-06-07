import { useState } from "react"
import { useTmaApp } from "@/shared/tma/context"
import type { Employee } from "@/entities/employee/types"
import { useFreeSlots } from "@/features/checker/queries"
import { CatBtn, CatIcon } from "@/shared/ui/cat/primitives"
import { CheckerFilters } from "../components/checker-filters"
import { CheckerPeopleSection } from "../components/checker-people-section"
import { CheckerSlotsResults } from "../components/checker-slots-results"

export function CheckerPage() {
  const t = useTmaApp().t
  const [people, setPeople] = useState<Employee[]>([])
  const [search, setSearch] = useState("")
  const [range, setRange] = useState("7")
  const [dur, setDur] = useState(60)

  const mutation = useFreeSlots()
  const slots = (mutation.data ?? []).filter((s) => s.mins >= dur)

  return (
    <div className="px-4 pb-7">
      <h2 className="tma-heading mx-1 mt-0.5 mb-1 text-[26px]">
        {t("findTime")} 🔍
      </h2>
      <p className="mx-1 mb-[18px] text-sm text-tma-muted">
        Кот найдёт окно, когда все свободны
      </p>

      <CheckerPeopleSection
        people={people}
        onChange={setPeople}
        search={search}
        onSearchChange={setSearch}
      />

      <CheckerFilters
        range={range}
        onRangeChange={setRange}
        dur={dur}
        onDurChange={setDur}
      />

      <div className="h-[22px]" />
      <CatBtn
        variant="primary"
        full
        size="lg"
        disabled={people.length === 0 || mutation.isPending}
        onClick={() => {
          const today = new Date()
          const from = today.toISOString().slice(0, 10)
          const toDate = new Date(today)
          toDate.setDate(toDate.getDate() + Number(range))
          const to = toDate.toISOString().slice(0, 10)
          mutation.mutate({
            participants: people.map((e) => e.email),
            from,
            to,
            durationMins: dur,
          })
        }}
        icon={<CatIcon name="search" size={20} sw={2.2} />}
      >
        {mutation.isPending ? "…" : t("findSlots")}
      </CatBtn>

      {mutation.isError && (
        <CheckerSlotsResults people={people} dur={dur} slots={[]} isError />
      )}

      {mutation.isSuccess && (
        <CheckerSlotsResults
          people={people}
          dur={dur}
          slots={slots}
          isError={false}
        />
      )}
    </div>
  )
}
