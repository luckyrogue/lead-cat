import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { api } from "@/shared/api/client"
import { useRequireWorkspace } from "@/shared/hooks/use-require-workspace"
import { useState } from "react"
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id"

type Member = {
  id: string
  telegram_username: string
  role: string
  github_login: string
  gitlab_login: string
}

export function TeamPage() {
  const workspaceId = useWorkspaceId()
  const ready = useRequireWorkspace()
  const [username, setUsername] = useState("")
  const [drafts, setDrafts] = useState<
    Record<string, { github: string; gitlab: string }>
  >({})
  const qc = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ["members", workspaceId],
    queryFn: async () =>
      (await api.get<Member[]>(`/workspaces/${workspaceId}/members`)).data,
    enabled: !!workspaceId,
  })

  const sync = useMutation({
    mutationFn: async () =>
      (
        await api.post<{ synced: number }>(
          `/workspaces/${workspaceId}/members/sync-chat`
        )
      ).data,
    onSuccess: (res) => {
      qc.invalidateQueries({ queryKey: ["members", workspaceId] })
      toast.success(`Добавлено из канала: ${res.synced}`)
    },
    onError: () => toast.error("Синхронизация не удалась"),
  })

  const add = useMutation({
    mutationFn: async () =>
      api.post(`/workspaces/${workspaceId}/members`, {
        telegram_username: username,
        role: "developer",
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["members", workspaceId] })
      setUsername("")
      toast.success("Котик добавлен")
    },
    onError: () => toast.error("Не удалось добавить"),
  })

  const saveVCS = useMutation({
    mutationFn: async (m: Member) => {
      const d = drafts[m.telegram_username] ?? {
        github: m.github_login,
        gitlab: m.gitlab_login,
      }
      await api.patch(
        `/workspaces/${workspaceId}/members/${m.telegram_username}/vcs`,
        {
          github_login: d.github,
          gitlab_login: d.gitlab,
        }
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["members", workspaceId] })
      toast.success("VCS сохранён")
    },
    onError: () => toast.error("Ошибка сохранения"),
  })

  if (!ready) return null
  if (isLoading) {
    return (
      <p className="text-muted-foreground flex items-center gap-2">
        <Spinner />
        Кот собирает команду…
      </p>
    )
  }

  const draft = (m: Member) =>
    drafts[m.telegram_username] ?? {
      github: m.github_login,
      gitlab: m.gitlab_login,
    }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">Котики команды</h2>
      <p className="text-muted-foreground text-sm">
        Бот должен быть админом в канале. Участники подтягиваются из админов и
        из тех, кто писал в чат.
      </p>
      <Button
        className="w-full"
        onClick={() => sync.mutate()}
        disabled={sync.isPending}
      >
        {sync.isPending ? <Spinner className="mr-2" /> : null}
        Синхронизировать из канала
      </Button>

      <ul className="space-y-3">
        {data?.map((m) => (
          <li
            key={m.id}
            className="border-border bg-card space-y-3 rounded-2xl border p-3"
          >
            <div className="flex justify-between text-sm">
              <span className="font-medium">@{m.telegram_username}</span>
              <span className="text-muted-foreground">{m.role}</span>
            </div>
            <div className="space-y-2">
              <Label>GitHub login</Label>
              <Input
                placeholder="octocat"
                value={draft(m).github}
                onChange={(e) =>
                  setDrafts((prev) => ({
                    ...prev,
                    [m.telegram_username]: {
                      ...draft(m),
                      github: e.target.value,
                    },
                  }))
                }
              />
            </div>
            <div className="space-y-2">
              <Label>GitLab login</Label>
              <Input
                placeholder="username"
                value={draft(m).gitlab}
                onChange={(e) =>
                  setDrafts((prev) => ({
                    ...prev,
                    [m.telegram_username]: {
                      ...draft(m),
                      gitlab: e.target.value,
                    },
                  }))
                }
              />
            </div>
            <Button
              variant="outline"
              className="w-full"
              size="sm"
              onClick={() => saveVCS.mutate(m)}
              disabled={saveVCS.isPending}
            >
              Сохранить VCS
            </Button>
          </li>
        ))}
      </ul>

      <div className="border-border bg-muted/40 space-y-3 rounded-2xl border p-4">
        <h3 className="text-sm font-semibold">Добавить вручную</h3>
        <div className="space-y-2">
          <Label htmlFor="tg-user">Telegram</Label>
          <Input
            id="tg-user"
            placeholder="@username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
          />
        </div>
        <Button
          className="w-full"
          onClick={() => add.mutate()}
          disabled={add.isPending}
        >
          Добавить кота
        </Button>
      </div>
    </div>
  )
}
