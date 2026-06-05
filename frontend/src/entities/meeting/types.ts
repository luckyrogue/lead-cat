import type { Employee } from "@/entities/employee/types"

export type Meeting = {
  id: string
  type: string
  dept: string
  host: string
  date: string
  start: string
  end: string
  rec: string
  recDays?: number[]
  organizer: string
  participants: string[]
  desc?: string
}

export type FreeSlot = {
  day: string
  iso: string
  start: string
  end: string
  mins: number
}

export type MeetingDraft = {
  dept: string
  type: string
  host: string
  date: string
  start: string
  dur: number
  rec: string
  recDays: number[]
  participants: Employee[]
  desc: string
  end?: string
}
