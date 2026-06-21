export type MeetingGroup<T extends { series_id?: string | null }> =
  | { kind: "single"; meeting: T }
  | { kind: "series"; seriesId: string; meetings: T[] }

export function groupBySeries<T extends { series_id?: string | null }>(
  meetings: T[]
): MeetingGroup<T>[] {
  const order: string[] = []
  const bySeries = new Map<string, T[]>()
  const singles: MeetingGroup<T>[] = []
  for (const m of meetings) {
    const sid = m.series_id ?? ""
    if (!sid) {
      singles.push({ kind: "single", meeting: m })
      continue
    }
    if (!bySeries.has(sid)) {
      bySeries.set(sid, [])
      order.push(sid)
    }
    bySeries.get(sid)!.push(m)
  }
  return [
    ...order.map(
      (sid): MeetingGroup<T> => ({
        kind: "series",
        seriesId: sid,
        meetings: bySeries.get(sid)!,
      })
    ),
    ...singles,
  ]
}
