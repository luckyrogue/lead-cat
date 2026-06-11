import { z } from "zod"

export const createMeetingSchema = z
  .object({
    type: z.string().trim().min(1, "Title is required"),
    dept: z.string(),
    date: z.string().min(1, "Date is required"),
    start: z.string().min(1, "Start time is required"),
    end: z.string().min(1, "End time is required"),
    recurrence: z.enum(["once", "daily", "weekly", "monthly", "custom"]),
    recurrence_until: z.string(),
    recurrence_days: z.array(z.number().int().min(1).max(7)),
    desc: z.string(),
  })
  .refine((v) => v.end > v.start, {
    message: "End must be after start",
    path: ["end"],
  })
  .refine((v) => v.recurrence === "once" || v.recurrence_until.length > 0, {
    message: "Pick an end date for a repeating meeting",
    path: ["recurrence_until"],
  })
  .refine((v) => v.recurrence !== "custom" || v.recurrence_days.length > 0, {
    message: "Pick at least one weekday",
    path: ["recurrence_days"],
  })

export type CreateMeetingForm = z.infer<typeof createMeetingSchema>
