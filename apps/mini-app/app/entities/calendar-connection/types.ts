export type CalendarProvider = "google"

export type CalendarConnection = {
  provider: CalendarProvider
  connected: boolean
  email: string
  scopes: string
}

export type StartConnectResponse = {
  auth_url: string
}
