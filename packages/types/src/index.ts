export * from "./meeting/weekdays";

export type Surface = "telegram" | "web";

export type OrgRole = "owner" | "admin" | "member";

export interface OrgMember {
  id: string;
  role: OrgRole;
  surface: Surface;
}
