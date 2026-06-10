import { lazy, Suspense } from "react";
import { Button, Card } from "@leadcat/ui";

const R3FScene = lazy(() =>
  import("@leadcat/ui/3d").then((m) => ({ default: m.R3FScene })),
);

export function App() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 bg-surface-muted p-6">
      <Card className="w-full max-w-md text-center">
        <h1 className="text-2xl font-bold text-brand-700">Lead Cat Admin</h1>
        <p className="mt-2 text-sm text-brand-500">Operator console placeholder.</p>
        <Button variant="soft" className="mt-5">
          Manage workspace
        </Button>
      </Card>
      <div className="h-64 w-full max-w-md overflow-hidden rounded-3xl shadow-soft">
        <Suspense fallback={<div className="grid h-full place-items-center text-brand-400">Loading 3D…</div>}>
          <R3FScene />
        </Suspense>
      </div>
    </main>
  );
}
