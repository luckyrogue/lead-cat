import type { Dict } from "~/shared/i18n/types"

export const en: Dict = {
  common: {
    save: "Save",
    cancel: "Cancel",
    delete: "Delete",
    loading: "Loading…",
    retry: "Retry",
    back: "Back",
  },
  nav: {
    home: "Home",
    meetings: "Meetings",
    checker: "Checker",
    profile: "Profile",
  },
  home: {
    appSubtitle: "Lead Cat",
    greeting: "Hi, {name}",
    bookMeeting: "Book a meeting",
    upcomingTitle: "Upcoming",
    seeAll: "See all",
    errorLoad: "Couldn't load meetings",
    emptyTitle: "No upcoming meetings",
    emptyHint: "Book one to get started.",
  },
  states: {
    tryAgain: "Try again",
  },
  auth: {
    signingIn: "Signing you in…",
    errorTitle: "Couldn't sign in",
    errorMissingInitData: "Open this app from inside Telegram to continue.",
    errorGeneric: "Something went wrong while authenticating.",
    notRegisteredTitle: "Not registered yet",
    notRegisteredHint:
      "Open the bot and press start to get access to your meetings.",
    openBot: "Open the bot",
    contactAdmin: "Contact your administrator to get registered.",
  },
}
