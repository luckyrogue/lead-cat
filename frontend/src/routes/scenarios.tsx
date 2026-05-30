import { createFileRoute, Link } from "@tanstack/react-router"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import {
  createScenario,
  fetchScenarios,
  runScenario,
} from "@/entities/scenario/api"
import { presetCommits } from "@/shared/presets"
import { ScenarioEditor } from "@/features/scenario-editor/scenario-editor"
import { useRequireWorkspace } from "@/shared/hooks/use-require-workspace"
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id"

export const Route = createFileRoute("/scenarios")({
  validateSearch: (s: Record<string, unknown>) => ({
    workspaceId: (s.workspaceId as string) || "",
  }),
  component: ScenariosPage,
})

function ScenariosPage() {
  const workspaceId = useWorkspaceId()
  const ready = useRequireWorkspace()
  const [editId, setEditId] = useState<string | null>(null)
  const qc = useQueryClient()
  const { data } = useQuery({
    queryKey: ["scenarios", workspaceId],
    queryFn: () => fetchScenarios(workspaceId),
    enabled: !!workspaceId,
  })
  const run = useMutation({
    mutationFn: (sid: string) => runScenario(workspaceId, sid),
    onSuccess: () => toast.success("Сценарий запущен"),
    onError: () => toast.error("Не удалось запустить"),
  })
  const create = useMutation({
    mutationFn: async () => {
      const sc = await createScenario(workspaceId, {
        name: `Сценарий ${(data?.length ?? 0) + 1}`,
        definition: presetCommits,
      })
      return sc.id
    },
    onSuccess: (id) => {
      qc.invalidateQueries({ queryKey: ["scenarios", workspaceId] })
      setEditId(id)
    },
    onError: () => toast.error("Не удалось создать сценарий"),
  })

  if (!ready) return null
  if (editId) {
    return (
      <ScenarioEditor
        workspaceId={workspaceId}
        scenarioId={editId}
        onBack={() => setEditId(null)}
      />
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-xl font-bold">Собери цепочку мяу</h2>
        <Button
          size="sm"
          className="shrink-0"
          onClick={() => create.mutate()}
          disabled={create.isPending}
        >
          {create.isPending ? <Spinner className="mr-2" /> : null}+ Новый
          сценарий
        </Button>
      </div>
      {data?.map((s) => (
        <div
          key={s.id}
          className="border-border bg-card rounded-2xl border p-4"
        >
          <div className="font-semibold">{s.name}</div>
          <div className="text-muted-foreground text-xs">
            {s.enabled ? "включён" : "спит"}
          </div>
          <div className="mt-2 flex gap-2">
            <Button
              size="sm"
              onClick={() => run.mutate(s.id)}
              disabled={run.isPending}
            >
              ▶ Прогнать
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setEditId(s.id)}>
              Редактировать
            </Button>
          </div>
        </div>
      ))}
      <Button variant="link" className="h-auto p-0" asChild>
        <Link
          to="/scenarios"
          search={{ workspaceId }}
          onClick={() =>
            qc.invalidateQueries({ queryKey: ["scenarios", workspaceId] })
          }
        >
          Обновить
        </Link>
      </Button>
    </div>
  )
}
