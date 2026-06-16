import type { Dict } from "~/shared/i18n/types"

export const en: Dict = {
  common: {
    save: "Save",
    cancel: "Cancel",
    delete: "Delete",
    edit: "Edit",
    loading: "Loading…",
    retry: "Retry",
    back: "Back",
    saved: "Saved",
    saveFailed: "Save failed",
    failedToLoad: "Failed to load.",
  },
  nav: {
    dashboard: "Dashboard",
    members: "Members",
    invites: "Invites",
    meetings: "Meetings",
    settings: "Settings",
  },
  dashboard: {
    eyebrow: "Lead Cat Admin",
    welcome: "Welcome, {name}",
    descriptionOrg: "A calm snapshot of {org}.",
    descriptionDefault: "A calm snapshot of your workspace.",
    statOrganization: "Organization",
    statOrganizationHintNone: "No organization yet",
    statOrganizationHintSlug: "Slug: {slug}",
    statMembers: "Members",
    statMembersHint: "People with access to this organization",
    statInvites: "Pending invites",
    statInvitesHint: "Invitations awaiting acceptance",
  },
  sidebar: {
    selectOrganization: "Select organization",
    organizationLabel: "Organization",
    organizationsMenuLabel: "Organizations",
    logoutLabel: "Log out",
    loadingWorkspace: "Loading your workspace…",
  },
}
