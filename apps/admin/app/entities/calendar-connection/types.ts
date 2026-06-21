import type { CalendarConnectionView } from "@leadcat/api-client"

export type CalendarProvider = CalendarConnectionView["provider"]

export type CalendarConnection = CalendarConnectionView

export type StartConnectResponse = {
  auth_url: string
}
