import { z } from "zod"

export const tmaAuthRequestSchema = z.object({
  init_data: z.string().min(1),
})

export type TmaAuthRequest = z.infer<typeof tmaAuthRequestSchema>
