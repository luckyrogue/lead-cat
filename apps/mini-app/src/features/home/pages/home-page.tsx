import { Button, Card, CatHead, Paw } from "@leadcat/ui";
import type { Surface } from "@leadcat/types";

const surface: Surface = "telegram";

export function HomePage() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-gradient-to-b from-cream-100 to-peach-50 p-6">
      <Card bordered className="max-w-sm animate-pop text-center">
        <div className="mb-4 flex justify-center">
          <span className="grid h-16 w-16 place-items-center rounded-2xl bg-coral-100 text-coral-500">
            <CatHead className="h-9 w-9" />
          </span>
        </div>
        <h1 className="text-2xl font-bold text-kitty-800">Lead Cat</h1>
        <p className="mt-2 text-sm text-kitty-600">Mini App ({surface}).</p>
        <Button className="mt-5">
          <Paw className="h-5 w-5" />
          Get started
        </Button>
      </Card>
    </main>
  );
}
