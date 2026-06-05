import type { Employee } from "@/entities/employee/types"
import { useTmaApp } from "@/shared/tma/context"
import { Avatar, CatIcon, inputStyle } from "@/shared/ui/cat/primitives"

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
  const p = useTmaApp()

  const add = (e: Employee) => {
    onChange([...value, e])
    onSearchChange("")
  }

  const remove = (index: number) => {
    onChange(value.filter((_, j) => j !== index))
  }

  return (
    <>
      <div style={{ position: "relative" }}>
        <input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={searchPlaceholder}
          style={{ ...inputStyle(p), paddingLeft: 42 }}
        />
        <CatIcon
          name="search"
          size={19}
          color={p.faint}
          style={{ position: "absolute", left: 13, top: 15 }}
          sw={2}
        />
      </div>
      {matches.length > 0 && (
        <div
          style={{
            marginTop: 8,
            background: p.card,
            borderRadius: 14,
            border: `1px solid ${p.border}`,
            overflow: "hidden",
          }}
        >
          {matches.map((e, i) => (
            <button
              key={e.id}
              type="button"
              onClick={() => add(e)}
              style={{
                width: "100%",
                display: "flex",
                alignItems: "center",
                gap: 11,
                padding: "10px 12px",
                border: "none",
                borderBottom:
                  i < matches.length - 1 ? `1px solid ${p.border}` : "none",
                background: "transparent",
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              <Avatar name={e.name} size={34} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div
                  style={{
                    fontWeight: 700,
                    fontSize: 14,
                    color: p.text,
                    fontFamily: "var(--font-display)",
                  }}
                >
                  {e.name}
                </div>
                <div style={{ fontSize: 12, color: p.muted }}>
                  {showEmail ? `${e.dept} · ${e.email}` : e.dept}
                </div>
              </div>
              <CatIcon name="plus" size={18} color={p.accent} sw={2.4} />
            </button>
          ))}
        </div>
      )}
      {value.length > 0 && (
        <div
          style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 12 }}
        >
          {value.map((e, i) => (
            <span
              key={e.id}
              style={{
                display: "inline-flex",
                alignItems: "center",
                gap: 7,
                background: p.cardAlt,
                borderRadius: 999,
                padding: "5px 6px 5px 5px",
                border: `1px solid ${p.border}`,
              }}
            >
              <Avatar name={e.name} size={24} />
              <span
                style={{
                  fontSize: 13,
                  fontWeight: 700,
                  color: p.text,
                  fontFamily: "var(--font-display)",
                }}
              >
                {e.name.split(" ")[0]}
              </span>
              <button
                type="button"
                onClick={() => remove(i)}
                style={{
                  border: "none",
                  background: "none",
                  cursor: "pointer",
                  display: "flex",
                  padding: 2,
                }}
              >
                <CatIcon name="x" size={14} color={p.muted} sw={2.4} />
              </button>
            </span>
          ))}
        </div>
      )}
    </>
  )
}
