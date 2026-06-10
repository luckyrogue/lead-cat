import { Badge, Input, Loader2, Search, X } from "@leadcat/ui"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"

import { employeeSearchQuery } from "~/entities/employee/queries"
import type { Employee } from "~/entities/employee/types"

type EmployeePickerProps = {
  selected: Employee[]
  onChange: (next: Employee[]) => void
}

export function EmployeePicker({ selected, onChange }: EmployeePickerProps) {
  const [query, setQuery] = useState("")
  const trimmed = query.trim()
  const search = useQuery(employeeSearchQuery(trimmed))

  const selectedEmails = new Set(selected.map((e) => e.email))
  const results = (search.data ?? []).filter((e) => !selectedEmails.has(e.email))

  function add(emp: Employee) {
    onChange([...selected, emp])
    setQuery("")
  }

  function remove(email: string) {
    onChange(selected.filter((e) => e.email !== email))
  }

  return (
    <div className="flex flex-col gap-2">
      {selected.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {selected.map((emp) => (
            <Badge key={emp.email} tone="muted" className="gap-1 pr-1">
              {emp.name || emp.email}
              <button
                type="button"
                onClick={() => remove(emp.email)}
                className="rounded-full p-0.5 hover:bg-foreground/10"
                aria-label={`Remove ${emp.name || emp.email}`}
              >
                <X className="size-3" />
              </button>
            </Badge>
          ))}
        </div>
      ) : null}

      <div className="relative">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search employees…"
          className="pl-9"
        />
        {search.isFetching ? (
          <Loader2 className="absolute right-3 top-1/2 size-4 -translate-y-1/2 animate-spin text-muted-foreground" />
        ) : null}
      </div>

      {trimmed.length > 0 ? (
        <div className="overflow-hidden rounded-[calc(var(--radius)*0.7)] border border-border/70">
          {results.length === 0 && !search.isFetching ? (
            <p className="px-3 py-2.5 text-sm text-muted-foreground">No matches</p>
          ) : (
            <ul className="divide-y divide-border/60">
              {results.map((emp) => (
                <li key={emp.email}>
                  <button
                    type="button"
                    onClick={() => add(emp)}
                    className="flex w-full flex-col items-start px-3 py-2.5 text-left hover:bg-muted/50"
                  >
                    <span className="text-sm font-medium text-foreground">
                      {emp.name || emp.email}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {emp.email}
                      {emp.dept ? ` · ${emp.dept}` : ""}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}
    </div>
  )
}
