export type Scenario = {
  id: string
  name: string
  enabled: boolean
  trigger: { hour: number; minute: number; days: number[] }
  actions: string[]
  note: string
}
