import type { Employee } from "@/entities/employee/types"
import { cn } from "@/shared/lib/cn"
import { Avatar, CatIcon } from "@/shared/ui/cat/primitives"

export function EmployeePicker({
  value,
  onChange,
  search,
  onSearchChange,
  matches,
  searchPlaceholder,
  showEmail = false,
}: {
  value: Employee[]
  onChange: (next: Employee[]) => void
  search: string
  onSearchChange: (q: string) => void
  matches: Employee[]
  searchPlaceholder: string
  showEmail?: boolean
}) {
  const add = (e: Employee) => {
    onChange([...value, e])
    onSearchChange("")
  }

  const remove = (index: number) => {
    onChange(value.filter((_, j) => j !== index))
  }

  return (
    <>
      <div className="relative">
        <input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={searchPlaceholder}
          className={cn("tma-input pl-[42px]")}
        />
        <span className="pointer-events-none absolute left-[13px] top-[15px] text-tma-faint">
          <CatIcon name="search" size={19} sw={2} />
        </span>
      </div>
      {matches.length > 0 && (
        <div className="mt-2 overflow-hidden rounded-[14px] border border-tma-border bg-tma-card">
          {matches.map((e, i) => (
            <button
              key={e.id}
              type="button"
              onClick={() => add(e)}
              className={cn(
                "flex w-full cursor-pointer items-center gap-[11px] border-none bg-transparent px-3 py-2.5 text-left",
                i < matches.length - 1 && "border-b border-tma-border"
              )}
            >
              <Avatar name={e.name} size={34} />
              <div className="min-w-0 flex-1">
                <div className="font-display text-sm font-bold text-tma-text">
                  {e.name}
                </div>
                <div className="text-xs text-tma-muted">
                  {showEmail ? `${e.dept} · ${e.email}` : e.dept}
                </div>
              </div>
              <span className="text-tma-accent">
                <CatIcon name="plus" size={18} sw={2.4} />
              </span>
            </button>
          ))}
        </div>
      )}
      {value.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {value.map((e, i) => (
            <span
              key={e.id}
              className="inline-flex items-center gap-[7px] rounded-full border border-tma-border bg-tma-card-alt py-[5px] pr-1.5 pl-[5px]"
            >
              <Avatar name={e.name} size={24} />
              <span className="font-display text-[13px] font-bold text-tma-text">
                {e.name.split(" ")[0]}
              </span>
              <button
                type="button"
                onClick={() => remove(i)}
                className="flex cursor-pointer border-none bg-transparent p-0.5 text-tma-muted"
              >
                <CatIcon name="x" size={14} sw={2.4} />
              </button>
            </span>
          ))}
        </div>
      )}
    </>
  )
}
