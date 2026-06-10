import { z } from "zod"

export const createMeetingSchema = z
  .object({
    type: z.string().trim().min(1, "Title is required"),
    dept: z.string(),
    date: z.string().min(1, "Date is required"),
    start: z.string().min(1, "Start time is required"),
    end: z.string().min(1, "End time is required"),
    desc: z.string(),
  })
  .refine((v) => v.end > v.start, {
    message: "End must be after start",
    path: ["end"],
  })

export type CreateMeetingForm = z.infer<typeof createMeetingSchema>
