import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  addEdge,
  useEdgesState,
  useNodesState,
  type Connection,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { scenarioPresets } from "@/shared/presets";
import { fetchScenario, updateScenario } from "@/entities/scenario/api";

const nodeTypes = [
  { type: "trigger.cron", label: "Cron", color: "#f97316" },
  { type: "trigger.manual", label: "Manual", color: "#fb923c" },
  { type: "action.telegram.message", label: "Message", color: "#fdba74" },
  { type: "action.telegram.cat_photo", label: "Cat photo", color: "#fed7aa" },
  { type: "action.vcs.commits_report", label: "Commits", color: "#ffedd5" },
] as const;

type DefNode = { id: string; type: string; parameters?: Record<string, unknown> };
type ScenarioDef = {
  nodes?: readonly DefNode[];
  edges?: readonly { source: string; target: string }[];
};

function toFlow(def: ScenarioDef) {
  const nodes: Node[] = (def.nodes ?? []).map((n, i) => {
    const meta = nodeTypes.find((t) => t.type === n.type);
    return {
      id: n.id,
      type: "default",
      position: { x: 40 + (i % 3) * 180, y: 40 + Math.floor(i / 3) * 100 },
      data: { label: meta?.label ?? n.type, type: n.type, parameters: n.parameters ?? {} },
      style: { border: `2px solid ${meta?.color ?? "#f97316"}`, borderRadius: 12, padding: 8 },
    };
  });
  const edges: Edge[] = (def.edges ?? []).map((e, i) => ({
    id: `e-${i}`,
    source: e.source,
    target: e.target,
  }));
  return { nodes, edges };
}

function fromFlow(nodes: Node[], edges: Edge[]) {
  return {
    nodes: nodes.map((n) => ({
      id: n.id,
      type: (n.data as { type?: string }).type ?? "action.telegram.message",
      parameters: (n.data as { parameters?: Record<string, unknown> }).parameters ?? {},
    })),
    edges: edges.map((e) => ({ source: e.source, target: e.target })),
  };
}

type Props = {
  workspaceId: string;
  scenarioId: string;
  onBack: () => void;
};

export function ScenarioEditor({ workspaceId, scenarioId, onBack }: Props) {
  const [name, setName] = useState("");
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    fetchScenario(workspaceId, scenarioId).then((sc) => {
      setName(sc.name);
      const flow = toFlow(
        sc.definition as {
          nodes: Array<{ id: string; type: string }>;
          edges: Array<{ source: string; target: string }>;
        },
      );
      setNodes(flow.nodes);
      setEdges(flow.edges);
    });
  }, [workspaceId, scenarioId, setNodes, setEdges]);

  const onConnect = useCallback(
    (c: Connection) => setEdges((eds) => addEdge(c, eds)),
    [setEdges],
  );

  const addNode = (type: string, label: string) => {
    const id = `n-${Date.now()}`;
    setNodes((ns) => [
      ...ns,
      {
        id,
        type: "default",
        position: { x: 80, y: 80 + ns.length * 60 },
        data: { label, type, parameters: {} },
        style: { border: "2px solid var(--primary)", borderRadius: 12, padding: 8 },
      },
    ]);
  };

  const applyPreset = (def: ScenarioDef) => {
    const flow = toFlow(def);
    setNodes(flow.nodes);
    setEdges(flow.edges);
  };

  const save = async () => {
    setSaving(true);
    try {
      await updateScenario(workspaceId, scenarioId, {
        name,
        definition: fromFlow(nodes, edges),
      });
      toast.success("Сценарий сохранён");
    } catch {
      toast.error("Не удалось сохранить");
    } finally {
      setSaving(false);
    }
  };

  const palette = useMemo(() => nodeTypes, []);

  return (
    <div className="flex h-[calc(100vh-8rem)] flex-col space-y-3">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="sm" onClick={onBack}>
          ← Назад
        </Button>
        <Input className="flex-1" value={name} onChange={(e) => setName(e.target.value)} />
        <Button size="sm" onClick={save} disabled={saving}>
          {saving ? <Spinner className="mr-2" /> : null}
          Сохранить
        </Button>
      </div>
      <div className="flex flex-wrap items-center gap-1">
        <span className="w-full text-xs text-muted-foreground sm:w-auto">Пресеты:</span>
        {scenarioPresets.map((p) => (
          <Button
            key={p.id}
            variant="outline"
            size="xs"
            onClick={() => applyPreset(p.definition)}
          >
            {p.label}
          </Button>
        ))}
      </div>
      <div className="flex flex-wrap gap-1">
        {palette.map((p) => (
          <Button key={p.type} variant="secondary" size="xs" onClick={() => addNode(p.type, p.label)}>
            + {p.label}
          </Button>
        ))}
      </div>
      <div className="flex-1 overflow-hidden rounded-2xl border border-border bg-card">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          fitView
        >
          <Background />
          <Controls />
          <MiniMap />
        </ReactFlow>
      </div>
    </div>
  );
}
