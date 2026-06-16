import { useEffect, useRef } from "react"
import gsap from "gsap"
import { ScrollTrigger } from "gsap/ScrollTrigger"

// This module is imported by SSR-rendered components, so the plugin must only
// register in the browser.
if (typeof window !== "undefined") {
  gsap.registerPlugin(ScrollTrigger)
}

export { gsap, ScrollTrigger }

/**
 * Runs `build` once on mount, scoped to the returned ref element, and ONLY when
 * the user has not asked for reduced motion. `gsap.matchMedia` skips the body
 * entirely for `prefers-reduced-motion: reduce` and reverts every tween/
 * ScrollTrigger created inside it on unmount — so the markup degrades to a
 * static, fully-visible page with zero cleanup wiring per component.
 */
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
    // A function returned by `build` (e.g. to remove a manual listener) is
    // returned to gsap.matchMedia, which calls it during revert.
    mm.add("(prefers-reduced-motion: no-preference)", () =>
      buildRef.current(scope)
    )
    return () => mm.revert()
  }, [])

  return ref
}

// Touch devices get lighter motion: skip pointer parallax, soften scrubbing.
export function prefersCoarsePointer(): boolean {
  return (
    typeof window !== "undefined" &&
    window.matchMedia("(pointer: coarse)").matches
  )
}
