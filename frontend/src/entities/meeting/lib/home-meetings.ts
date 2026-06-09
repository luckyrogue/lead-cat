import type { Meeting } from "@/entities/meeting/types"

export function splitHomeMeetings(meetings: Meeting[], today: string) {
  const todayMeetings = meetings
    .filter((m) => m.date === today)
    .sort((a, b) => a.start.localeCompare(b.start))
  const upcomingMeetings = meetings
    .filter((m) => m.date > today)
    .sort((a, b) => (a.date + a.start).localeCompare(b.date + b.start))
    .slice(0, 4)
  return { todayMeetings, upcomingMeetings }
}
