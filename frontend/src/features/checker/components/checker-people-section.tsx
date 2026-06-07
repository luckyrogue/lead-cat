import type { Employee } from "@/entities/employee/types"
import { useTmaApp } from "@/shared/tma/context"
import { EmployeePicker } from "@/components/employee-picker"
import { Field } from "@/shared/ui/cat/primitives"
import { useEmployeeSearch } from "@/entities/employee/queries"

export function CheckerPeopleSection({
  people,
  onChange,
  search,
  onSearchChange,
}: {
  people: Employee[]
  onChange: (next: Employee[]) => void
  search: string
  onSearchChange: (q: string) => void
}) {
  const t = useTmaApp().t
  const { data: searchResults = [] } = useEmployeeSearch(search)
  const matches = searchResults
    .filter((e) => !people.find((x) => x.id === e.id))
    .slice(0, 5)

  return (
    <Field label={`${t("addPeople")} · ${people.length}`}>
      <EmployeePicker
        value={people}
        onChange={onChange}
        search={search}
        onSearchChange={onSearchChange}
        matches={matches}
        searchPlaceholder={t("searchPeople")}
      />
    </Field>
  )
}
