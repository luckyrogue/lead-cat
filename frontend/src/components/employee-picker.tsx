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
          className={cn("miniapp-input pl-[42px]")}
        />
        <span className="text-miniapp-faint pointer-events-none absolute left-[13px] top-[15px]">
          <CatIcon name="search" size={19} sw={2} />
        </span>
      </div>
      {matches.length > 0 && (
        <div className="border-miniapp-border bg-miniapp-card mt-2 overflow-hidden rounded-[14px] border">
          {matches.map((e, i) => (
            <button
              key={e.id}
              type="button"
              onClick={() => add(e)}
              className={cn(
                "flex w-full cursor-pointer items-center gap-[11px] border-none bg-transparent px-3 py-2.5 text-left",
                i < matches.length - 1 && "border-miniapp-border border-b"
              )}
            >
              <Avatar name={e.name} size={34} />
              <div className="min-w-0 flex-1">
                <div className="font-display text-miniapp-text text-sm font-bold">
                  {e.name}
                </div>
                <div className="text-miniapp-muted text-xs">
                  {showEmail ? `${e.dept} · ${e.email}` : e.dept}
                </div>
              </div>
              <span className="text-miniapp-accent">
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
              className="border-miniapp-border bg-miniapp-card-alt inline-flex items-center gap-[7px] rounded-full border py-[5px] pl-[5px] pr-1.5"
            >
              <Avatar name={e.name} size={24} />
              <span className="font-display text-miniapp-text text-[13px] font-bold">
                {e.name.split(" ")[0]}
              </span>
              <button
                type="button"
                onClick={() => remove(i)}
                className="text-miniapp-muted flex cursor-pointer border-none bg-transparent p-0.5"
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
