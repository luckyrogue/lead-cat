import { useQuery } from "@tanstack/react-query";
import { api } from "@/shared/api/client";
import { useRequireWorkspace } from "@/shared/hooks/use-require-workspace";
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id";

export function DashboardPage() {
  const workspaceId = useWorkspaceId();
  const ready = useRequireWorkspace();
  const { data } = useQuery({
    queryKey: ["workspace", workspaceId],
    queryFn: async () => (await api.get(`/workspaces/${workspaceId}`)).data,
    enabled: !!workspaceId,
  });
  if (!ready) return null;
  const w = data as { name?: string; notify_chat_id?: number } | undefined;
  return (
    <div className="space-y-3">
      <h2 className="text-xl font-bold">Лид-кот на посту</h2>
      <p>{w?.name}</p>
      <p className="text-sm">
        Чат: {w?.notify_chat_id ? `привязан (${w.notify_chat_id})` : "кот ещё не в группе"}
      </p>
    </div>
  );
}
