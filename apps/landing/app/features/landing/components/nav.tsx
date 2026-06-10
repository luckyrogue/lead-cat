import { Button, Paw } from "@leadcat/ui"

export function Nav() {
  return (
    <header className="relative z-20 mx-auto flex max-w-6xl items-center justify-between px-6 py-6">
      <a href="/" className="flex items-center gap-2 font-bold text-kitty-800">
        <span className="grid size-9 place-items-center rounded-2xl bg-coral-400 text-white">
          <Paw className="size-5" />
        </span>
        Lead Cat
      </a>
      <nav className="hidden items-center gap-8 text-sm font-semibold text-kitty-600 md:flex">
        <a className="transition-colors hover:text-coral-500" href="#features">
          Features
        </a>
        <a className="transition-colors hover:text-coral-500" href="#how">
          How it works
        </a>
      </nav>
      <Button size="sm">Get started</Button>
    </header>
  )
}
