import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Link, useNavigate } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { toast } from "sonner"
import { setStoredWorkspaceId } from "@/shared/hooks/use-workspace-id"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { createWorkspace, fetchWorkspaces } from "@/entities/workspace/api"

export function WorkspacesPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { data, isLoading } = useQuery({
    queryKey: ["workspaces"],
    queryFn: fetchWorkspaces,
  })
  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const create = useMutation({
    mutationFn: () => createWorkspace(name, slug),
    onSuccess: (ws) => {
      qc.invalidateQueries({ queryKey: ["workspaces"] })
      setName("")
      setSlug("")
      setStoredWorkspaceId(ws.id)
      toast.success("Логово создано")
      navigate({ to: "/dashboard", search: { workspaceId: ws.id } })
    },
    onError: () => toast.error("Не удалось создать логово"),
  })

  useEffect(() => {
    if (data?.length === 1) {
      setStoredWorkspaceId(data[0].id)
    }
  }, [data])

  if (isLoading) {
    return (
      <p className="text-muted-foreground flex items-center gap-2">
        <Spinner />
        Кот ищет логова…
      </p>
    )
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">Логова 🐾</h2>
      <p className="text-muted-foreground text-sm">
        Пока тихо… или заведи новое логово
      </p>
      <ul className="space-y-2">
        {data?.map((w, index) => (
          <li key={w.id || `${w.slug}-${index}`}>
            <Link
              to="/dashboard"
              search={{ workspaceId: w.id }}
              onClick={() => setStoredWorkspaceId(w.id)}
              className="border-border bg-card hover:bg-muted/50 block rounded-2xl border p-3 shadow-sm transition-colors"
            >
              {w.name}{" "}
              <span className="text-muted-foreground text-xs">/{w.slug}</span>
            </Link>
          </li>
        ))}
      </ul>
      <div className="border-border bg-muted/40 space-y-3 rounded-2xl border p-4">
        <h3 className="font-semibold">Новое логово</h3>
        <div className="space-y-2">
          <Label htmlFor="ws-name">Название</Label>
          <Input
            id="ws-name"
            placeholder="Название"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="ws-slug">Slug</Label>
          <Input
            id="ws-slug"
            placeholder="slug"
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
          />
        </div>
        <Button
          className="w-full"
          onClick={() => create.mutate()}
          disabled={create.isPending}
        >
          {create.isPending ? <Spinner className="mr-2" /> : null}
          Мур, создать
        </Button>
      </div>
    </div>
  )
}
