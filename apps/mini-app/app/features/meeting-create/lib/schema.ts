import { z } from "zod"

export const createMeetingSchema = z
  .object({
    type: z.string().trim().min(1, "create.errors.titleRequired"),
    dept: z.string(),
    date: z.string().min(1, "create.errors.dateRequired"),
    start: z.string().min(1, "create.errors.startRequired"),
    end: z.string().min(1, "create.errors.endRequired"),
    recurrence: z.enum(["once", "daily", "weekly", "monthly", "custom"]),
    recurrence_until: z.string(),
    recurrence_days: z.array(z.number().int().min(1).max(7)),
    desc: z.string(),
  })
  .refine((v) => v.end > v.start, {
    message: "create.errors.endAfterStart",
    path: ["end"],
  })
  .refine((v) => v.recurrence === "once" || v.recurrence_until.length > 0, {
    message: "create.errors.repeatUntilRequired",
    path: ["recurrence_until"],
  })
  .refine((v) => v.recurrence !== "custom" || v.recurrence_days.length > 0, {
    message: "create.errors.weekdayRequired",
    path: ["recurrence_days"],
  })

export type CreateMeetingForm = z.infer<typeof createMeetingSchema>
