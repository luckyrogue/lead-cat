import { z } from "zod";

export const workspaceSchema = z.object({
  id: z.string().uuid(),
  slug: z.string(),
  name: z.string(),
  notify_chat_id: z.number().nullable().optional(),
  meet_link: z.string(),
  tz: z.string(),
  vcs_provider: z.string(),
  has_vcs_token: z.boolean().optional(),
});

export const scenarioSchema = z.object({
  id: z.string().uuid(),
  workspace_id: z.string().uuid(),
  name: z.string(),
  enabled: z.boolean(),
  definition: z.record(z.string(), z.unknown()),
});

export type Workspace = z.infer<typeof workspaceSchema>;
export type Scenario = z.infer<typeof scenarioSchema>;
