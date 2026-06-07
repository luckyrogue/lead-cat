export type TmaUser = {
  telegramId: number
  name: string
  email: string
  role: "user" | "admin"
}

export type TmaUserRole = TmaUser["role"]
