import { Button, Card } from "@leadcat/ui";
import type { Surface } from "@leadcat/types";

const surface: Surface = "telegram";

export function App() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-surface-soft p-6">
      <Card className="max-w-sm animate-pop text-center">
        <h1 className="text-2xl font-bold text-brand-700">Lead Cat</h1>
        <p className="mt-2 text-sm text-brand-500">Mini App placeholder ({surface}).</p>
        <Button className="mt-5">Get started</Button>
      </Card>
    </main>
  );
}
