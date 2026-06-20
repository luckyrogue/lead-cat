import { useEffect, useRef } from "react"
import gsap from "gsap"
import { ScrollTrigger } from "gsap/ScrollTrigger"

if (typeof window !== "undefined") {
  gsap.registerPlugin(ScrollTrigger)
}

export { gsap, ScrollTrigger }

export function useSceneMotion<T extends HTMLElement = HTMLDivElement>(
  build: (scope: T) => void | (() => void)
) {
  const ref = useRef<T>(null)
  const buildRef = useRef(build)
  buildRef.current = build

  useEffect(() => {
    const scope = ref.current
    if (!scope) {
      return
    }
    const mm = gsap.matchMedia()

    mm.add("(prefers-reduced-motion: no-preference)", () =>
      buildRef.current(scope)
    )
    return () => mm.revert()
  }, [])

  return ref
}

export function prefersCoarsePointer(): boolean {
  return (
    typeof window !== "undefined" &&
    window.matchMedia("(pointer: coarse)").matches
  )
}
