import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { api } from "@/shared/api/client";
import { useRequireWorkspace } from "@/shared/hooks/use-require-workspace";
import { useWorkspaceId } from "@/shared/hooks/use-workspace-id";

type Integrations = {
  vcs_provider: string;
  vcs_namespace: string;
  vcs_base_url?: string;
  has_vcs_token: boolean;
  meet_link: string;
  tz: string;
};

export function IntegrationsPage() {
  const workspaceId = useWorkspaceId();
  const ready = useRequireWorkspace();
  const qc = useQueryClient();
  const { data } = useQuery({
    queryKey: ["integrations", workspaceId],
    queryFn: async () => (await api.get<Integrations>(`/workspaces/${workspaceId}/integrations`)).data,
    enabled: !!workspaceId,
  });
  const [provider, setProvider] = useState("github");
  const [token, setToken] = useState("");
  const [org, setOrg] = useState("Jaryq-Lab");
  const [baseURL, setBaseURL] = useState("https://gitlab.com");
  const [meet, setMeet] = useState("");
  const [tz, setTz] = useState("Asia/Almaty");

  useEffect(() => {
    if (!data) return;
    setProvider(data.vcs_provider || "github");
    setOrg(data.vcs_namespace || "");
    setBaseURL(data.vcs_base_url || "https://gitlab.com");
    setMeet(data.meet_link || "");
    setTz(data.tz || "Asia/Almaty");
  }, [data]);

  const save = useMutation({
    mutationFn: async () =>
      api.patch(`/workspaces/${workspaceId}/integrations`, {
        vcs_provider: provider,
        vcs_namespace: org,
        vcs_base_url: provider === "gitlab" ? baseURL : undefined,
        vcs_token: token || undefined,
        meet_link: meet,
        tz,
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["integrations", workspaceId] });
      setToken("");
      toast.success("Сохранено");
    },
    onError: () => toast.error("Не удалось сохранить"),
  });

  const verify = useMutation({
    mutationFn: async () => api.post(`/workspaces/${workspaceId}/integrations/verify`),
    onSuccess: () => toast.success("VCS ок"),
    onError: () => toast.error("Ошибка проверки VCS"),
  });

  if (!ready) return null;

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">Интеграции</h2>
      {data?.has_vcs_token && (
        <p className="text-xs text-muted-foreground">Токен сохранён ({data.vcs_provider})</p>
      )}

      <div className="space-y-2">
        <Label>VCS</Label>
        <Select value={provider} onValueChange={setProvider}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="github">GitHub</SelectItem>
            <SelectItem value="gitlab">GitLab</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label htmlFor="vcs-token">Token</Label>
        <Input
          id="vcs-token"
          type="password"
          placeholder={data?.has_vcs_token ? "новый токен (опционально)" : "token"}
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="vcs-org">{provider === "gitlab" ? "Group path" : "Org"}</Label>
        <Input id="vcs-org" value={org} onChange={(e) => setOrg(e.target.value)} />
      </div>

      {provider === "gitlab" && (
        <div className="space-y-2">
          <Label htmlFor="gitlab-url">GitLab URL</Label>
          <Input id="gitlab-url" value={baseURL} onChange={(e) => setBaseURL(e.target.value)} />
        </div>
      )}

      <div className="space-y-2">
        <Label htmlFor="meet">Meet link</Label>
        <Input id="meet" value={meet} onChange={(e) => setMeet(e.target.value)} />
      </div>

      <div className="space-y-2">
        <Label htmlFor="tz">Timezone</Label>
        <Input id="tz" value={tz} onChange={(e) => setTz(e.target.value)} />
      </div>

      <Button className="w-full" onClick={() => save.mutate()} disabled={save.isPending}>
        {save.isPending ? <Spinner className="mr-2" /> : null}
        Мур, сохранить
      </Button>
      <Button
        variant="outline"
        className="w-full"
        onClick={() => verify.mutate()}
        disabled={verify.isPending}
      >
        {verify.isPending ? <Spinner className="mr-2" /> : null}
        Проверить VCS
      </Button>
    </div>
  );
}
