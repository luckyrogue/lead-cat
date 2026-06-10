import { lazy, Suspense } from "react"
import { Paw } from "@leadcat/ui"

import { ClientOnly } from "~/components/client-only"

const CatScene = lazy(() =>
  import("@leadcat/ui/3d").then((module) => ({ default: module.CatScene }))
)

function SceneFallback() {
  return (
    <div className="grid h-full w-full place-items-center">
      <div className="flex flex-col items-center gap-3 text-coral-400">
        <Paw className="size-14 animate-bounce-soft" />
        <span className="text-sm font-semibold">Waking the kitty...</span>
      </div>
    </div>
  )
}

export function CatStage() {
  return (
    <ClientOnly fallback={<SceneFallback />}>
      {() => (
        <Suspense fallback={<SceneFallback />}>
          <CatScene followPointer />
        </Suspense>
      )}
    </ClientOnly>
  )
}
