type TFn = (key: string) => string

export function recurrenceLabel(t: TFn, rec: string): string {
  if (!rec || rec === "once") {
    return ""
  }
  const key = `create.recurrence.${rec}`
  const mapped = t(key)
  return mapped !== key ? mapped : rec
}
