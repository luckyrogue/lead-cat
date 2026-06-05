import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { api } from "@/shared/api/client"

declare global {
  interface Window {
    Telegram?: { WebApp?: { initData?: string } }
  }
}

type MeResponse = {
  telegram_id?: number | null
}

/** Links platform_users.telegram_id from web admin (not TMA bot_users auth). */
export function LinkTelegramBanner() {
  const qc = useQueryClient()
  const initData = window.Telegram?.WebApp?.initData

  const me = useQuery({
    queryKey: ["me"],
    queryFn: async () => (await api.get<MeResponse>("/me")).data,
    enabled: !!initData,
  })

  const link = useMutation({
    mutationFn: async () => {
      if (!initData) throw new Error("no initData")
      await api.post("/me/link-telegram", null, {
        headers: { "X-Telegram-Init-Data": initData },
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["me"] })
      toast.success("Telegram привязан")
    },
    onError: () => toast.error("Не удалось привязать Telegram"),
  })

  if (!initData) return null
  if (me.isLoading) return null
  if (me.data?.telegram_id) return null

  return (
    <Button
      variant="outline"
      className="mb-2 w-full"
      onClick={() => link.mutate()}
      disabled={link.isPending}
    >
      {link.isPending ? <Spinner className="mr-2" /> : null}
      Привязать Telegram к аккаунту
    </Button>
  )
}
