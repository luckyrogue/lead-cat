import { LandingPage } from "~/features/landing/pages/landing-page"

export function meta() {
  return [
    { title: "Lead Cat - meetings your team will actually love" },
    {
      name: "description",
      content:
        "Lead Cat finds the purrfect time across everyone's calendars. Native Google Calendar sync and a cozy Telegram Mini App.",
    },
  ]
}

export default function Index() {
  return <LandingPage />
}
