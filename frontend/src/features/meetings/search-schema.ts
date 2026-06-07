import { z } from "zod"

export const meetingsSearchSchema = z.object({
  scope: z.enum(["upcoming", "past", "all"]).optional().catch("upcoming"),
  q: z.string().optional().catch(""),
  page: z.coerce.number().optional().catch(1),
  success: z.string().optional(),
})

export type MeetingsSearch = z.infer<typeof meetingsSearchSchema>
