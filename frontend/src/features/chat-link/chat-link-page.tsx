import { useMutation, useQuery } from "@tanstack/react-query"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { api } from "@/shared/api/client"
import { useRequireWorkspace } from "@/shared/hooks/use-require-workspace"
import { useState } from "react"
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id"

export function ChatLinkPage() {
  const workspaceId = useWorkspaceId()
  const ready = useRequireWorkspace()
  const [chatId, setChatId] = useState("")
  const status = useQuery({
    queryKey: ["chat", workspaceId],
    queryFn: async () =>
      (await api.get(`/workspaces/${workspaceId}/chat/status`)).data,
    enabled: !!workspaceId,
  })
  const link = useMutation({
    mutationFn: async () =>
      api.post(`/workspaces/${workspaceId}/chat/link`, {
        chat_id: Number(chatId),
      }),
    onSuccess: () => {
      status.refetch()
      toast.success("Чат привязан")
    },
    onError: () => toast.error("Не удалось привязать"),
  })

  if (!ready) return null
  const st = status.data as
    | { linked?: boolean; notify_chat_id?: number }
    | undefined

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">Привязать логово</h2>
      <p className="text-muted-foreground text-sm">
        1. Добавь бота в группу 2. /chatid 3. Вставь id сюда
      </p>
      <p className="text-sm">
        Статус:{" "}
        {status.isLoading
          ? "…"
          : st?.linked
            ? `🐾 ${st.notify_chat_id}`
            : "не привязан"}
      </p>
      <div className="space-y-2">
        <Label htmlFor="chat-id">Chat ID</Label>
        <Input
          id="chat-id"
          placeholder="chat id"
          value={chatId}
          onChange={(e) => setChatId(e.target.value)}
        />
      </div>
      <Button
        className="w-full"
        onClick={() => link.mutate()}
        disabled={link.isPending}
      >
        {link.isPending ? <Spinner className="mr-2" /> : null}
        Привязать
      </Button>
    </div>
  )
}
