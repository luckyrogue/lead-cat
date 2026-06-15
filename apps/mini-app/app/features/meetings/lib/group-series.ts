import type { Meeting } from "~/entities/meeting/types"

export type MeetingGroup =
  | { kind: "single"; meeting: Meeting }
  | { kind: "series"; seriesId: string; meetings: Meeting[] }

export function groupBySeries(meetings: Meeting[]): MeetingGroup[] {
  const order: string[] = []
  const bySeries = new Map<string, Meeting[]>()
  const singles: MeetingGroup[] = []
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
      (sid): MeetingGroup => ({
        kind: "series",
        seriesId: sid,
        meetings: bySeries.get(sid)!,
      })
    ),
    ...singles,
  ]
}
