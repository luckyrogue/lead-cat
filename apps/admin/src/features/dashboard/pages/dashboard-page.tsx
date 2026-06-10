import { Button, Card, CatHead, Sparkle } from "@leadcat/ui";

export function DashboardPage() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-6 bg-gradient-to-b from-cream-100 to-peach-50 p-6">
      <Card bordered className="w-full max-w-md text-center">
        <div className="mb-4 flex justify-center">
          <span className="grid h-16 w-16 place-items-center rounded-2xl bg-coral-100 text-coral-500">
            <CatHead className="h-9 w-9" />
          </span>
        </div>
        <h1 className="flex items-center justify-center gap-2 text-2xl font-bold text-kitty-800">
          Lead Cat Admin
          <Sparkle className="h-5 w-5 text-sunny-400" />
        </h1>
        <p className="mt-2 text-sm text-kitty-600">Operator console.</p>
        <Button variant="soft" className="mt-5">
          Manage workspace
        </Button>
      </Card>
    </main>
  );
}
