export type MiniAppUser = {
  telegramId: number
  name: string
  email: string
  role: "user" | "admin"
}

export type MiniAppUserRole = MiniAppUser["role"]
