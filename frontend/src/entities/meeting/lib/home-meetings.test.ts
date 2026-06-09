import { describe, expect, it } from "vitest"
import type { Meeting } from "@/entities/meeting/types"
import { splitHomeMeetings } from "./home-meetings"

const m = (id: string, date: string, start: string): Meeting => ({
  id,
  type: "sync",
  dept: "Eng",
  host: "Host",
  date,
  start,
  end: "11:00",
  rec: "once",
  organizer: "a@x.com",
  participants: [],
  desc: "",
})

describe("splitHomeMeetings", () => {
  it("splits today and next four upcoming days", () => {
    const meetings = [
      m("1", "2026-06-07", "10:00"),
      m("2", "2026-06-07", "14:00"),
      m("3", "2026-06-08", "09:00"),
      m("4", "2026-06-09", "09:00"),
      m("5", "2026-06-10", "09:00"),
      m("6", "2026-06-11", "09:00"),
      m("7", "2026-06-12", "09:00"),
    ]

    const { todayMeetings, upcomingMeetings } = splitHomeMeetings(
      meetings,
      "2026-06-07"
    )

    expect(todayMeetings.map((x) => x.id)).toEqual(["1", "2"])
    expect(upcomingMeetings.map((x) => x.id)).toEqual(["3", "4", "5", "6"])
  })
})
