import { lazy, Suspense, useEffect } from "react"

import { ClientOnly } from "~/components/client-only"

const CatScene = lazy(() =>
  import("@leadcat/ui/3d").then((module) => ({ default: module.CatScene }))
)

function StagePlaceholder() {
  return (
    <div className="grid h-full w-full place-items-center" aria-hidden>
      <div className="relative flex h-[72%] w-[72%] max-w-[18rem] items-end justify-center">
        <div className="absolute bottom-[16%] h-[36%] w-[90%] rounded-full bg-[#F6D2B6]/75" />
        <div className="absolute bottom-[20%] h-[30%] w-[78%] rounded-full bg-[#FBE3CF]/90 shadow-[0_24px_48px_-28px_oklch(0.55_0.08_55_/_0.45)]" />
      </div>
    </div>
  )
}

export function CatStage() {
  useEffect(() => {
    void import("@leadcat/ui/3d")
  }, [])

  return (
    <ClientOnly fallback={<StagePlaceholder />}>
      {() => (
        <Suspense fallback={<StagePlaceholder />}>
          <CatScene autoRotate followPointer />
        </Suspense>
      )}
    </ClientOnly>
  )
}
