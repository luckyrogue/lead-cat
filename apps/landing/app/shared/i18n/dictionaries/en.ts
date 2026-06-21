import type { Dict } from "~/shared/i18n/types"

export const en = {
  meta: {
    title: "Lead Cat — meetings your team will actually love",
    description:
      "Lead Cat finds the purrfect time across everyone's calendars. Telegram Mini App for the team and a web admin panel for org owners.",
    keywords:
      "meeting scheduler, google calendar, telegram mini app, admin panel, team scheduling, appointment booking, lead cat",
    ogLocale: "en_US",
  },
  lang: {
    switchLabel: "Language",
    en: "English",
    ru: "Русский",
    kk: "Қазақша",
  },
  nav: {
    features: "Features",
    howItWorks: "How it works",
    miniApp: "Mini App",
    adminPanel: "Admin panel",
    getStarted: "Get started",
  },
  hero: {
    badge: "Scheduling, but cuddly",
    titleLine1: "Meetings your team will",
    titleHighlight: "actually love",
    copy: "Lead Cat sniffs out the purrfect time across everyone's calendars — no more back-and-forth, no more double-booking. Just happy little meetings.",
    ctaPrimary: "Get started — it's free",
    ctaSecondary: "See how it works",
    footnote:
      "Telegram Mini App for the team · web admin panel in the browser. Google Calendar. No credit card.",
    adminPanelLink: "Open admin panel →",
  },
  trustMarquee: {
    ariaLabel: "What Lead Cat does",
    items: [
      "Google Calendar sync",
      "Telegram Mini App",
      "Web admin panel",
      "Zero double-bookings",
      "Round-robin hosts",
      "Gentle reminders",
      "3-second booking",
      "Pooled availability",
      "Reschedules itself",
    ],
  },
  features: {
    title: "Everything you need to herd the cats",
    subtitle:
      "Powerful scheduling under the hood, wrapped in something genuinely delightful to use.",
    items: [
      {
        title: "One calm calendar view",
        body: "Lead Cat merges every calendar into one tidy availability map — Google, work, personal. See open paws at a glance.",
      },
      {
        title: "Reminders that don't nag",
        body: "Gentle, smart nudges right when they matter. Your team shows up on time without a single grumpy meow.",
      },
      {
        title: "Team scheduling, sorted",
        body: "Pick a group, and Lead Cat pounces on the slot that works for everyone. Round-robin and pooled hosts included.",
      },
      {
        title: "Lives where you work",
        body: "Telegram Mini App for daily booking and a web admin panel for orgs, members, and meetings. Google Calendar sync on both.",
      },
      {
        title: "Never lose a declined lead",
        body: "When a time doesn't fit, Lead Cat offers your visitor a short survey you built — capture the why, the feedback, and the lead, automatically.",
      },
    ],
  },
  howItWorks: {
    badge: "How it works",
    title: "Three little steps, zero headaches",
    steps: [
      {
        title: "Connect your calendars",
        body: "Link Google in two taps. Lead Cat learns when you're free.",
      },
      {
        title: "Share your kitty link",
        body: "Drop one link in chat. Guests pick a slot that just works.",
      },
      {
        title: "Show up and relax",
        body: "Auto-invites, reminders and reschedules. You do nothing.",
      },
    ],
    stats: [
      { to: 3, suffix: " sec", label: "to book a meeting" },
      { to: 0, suffix: "", label: "double-bookings" },
      { to: 100, suffix: "%", label: "more purring" },
    ],
  },
  showcase: {
    badge: "Right inside Telegram",
    title: "Your whole schedule, one cozy tap away",
    copy: "No new app to learn. Lead Cat lives in the Mini App your team already opens every day.",
    perks: [
      "Book, reschedule and confirm without leaving chat",
      "Tap a slot — invites and reminders send themselves",
      "Your calendar, your colleagues, one cozy place",
    ],
    today: "Today",
    dateLabel: "Tue, 16",
    meetings: [
      { time: "10:00", title: "Design sync" },
      { time: "13:30", title: "1:1 with Mia" },
      { time: "16:00", title: "Sprint demo" },
    ],
    bookedIn: "Booked in 3 taps",
  },
  footer: {
    headingWords: ["Ready", "to", "make", "meetings", "purr?"],
    copy: "Join the teams that traded scheduling chaos for a happy little cat.",
    cta: "Adopt Lead Cat",
    copyright: "Made with whiskers and warmth.",
  },
  errors: {
    genericTitle: "Something went wrong",
    genericDescription: "An unexpected error occurred.",
    notFoundDescription: "This page could not be found.",
    reload: "Reload",
  },
} satisfies Dict

export type LandingDict = typeof en
