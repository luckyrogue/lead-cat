export type CalendarConnection = {
  provider: "google" | "microsoft"
  connected: boolean
  email: string
  scopes: string
}
