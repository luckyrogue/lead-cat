import { apiFetch } from "@/shared/api/client"
import type { FreeSlot } from "@/entities/meeting/types"
import { fmtDate, type Lang } from "@/entities/meeting/lib/format"

type FreeSlotDTO = { iso: string; start: string; end: string; mins: number }

export type FreeSlotsParams = {
  participants: string[]
  from: string
  to: string
  durationMins: number
}

export async function fetchFreeSlots(
  params: FreeSlotsParams,
  lang: Lang
): Promise<FreeSlot[]> {
  const data = await apiFetch<{ slots: FreeSlotDTO[] }>("/tma/free-slots", {
    method: "POST",
    body: {
      participants: params.participants,
      from: params.from,
      to: params.to,
      duration_mins: params.durationMins,
    },
  })
  return data.slots.map((s) => ({
    day: fmtDate(s.iso, lang),
    iso: s.iso,
    start: s.start,
    end: s.end,
    mins: s.mins,
  }))
}

export type Conflict = {
  email: string
  name: string
  title: string
  start: string
  end: string
}

export type ConflictsParams = {
  participants: string[]
  date: string
  start: string
  end: string
  excludeId?: string
}

export async function fetchConflicts(
  params: ConflictsParams
): Promise<Conflict[]> {
  const data = await apiFetch<{ conflicts: Conflict[] }>("/tma/conflicts", {
    method: "POST",
    body: {
      participants: params.participants,
      date: params.date,
      start: params.start,
      end: params.end,
      exclude_id: params.excludeId ?? "",
    },
  })
  return data.conflicts
}
