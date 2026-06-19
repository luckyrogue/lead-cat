export type BookingEventType = {
  id: string
  host_user_id: string
  organization_id: string
  slug: string
  title: string
  description: string
  duration_mins: number
  active: boolean
  timezone: string
  avail_weekdays: number[]
  avail_start_minute: number
  avail_end_minute: number
  created_at: string
  updated_at: string
}

export type EventTypeInput = {
  title: string
  description: string
  duration_mins: number
  timezone: string
  avail_weekdays: number[]
  avail_start_minute: number
  avail_end_minute: number
  active: boolean
}
