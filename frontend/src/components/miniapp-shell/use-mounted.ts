import { useEffect, useState } from "react"

/** Keep mounted briefly on exit for exit animations */
export function useMounted(open: boolean, ms = 300) {
  const [mounted, setMounted] = useState(open)
  const [shown, setShown] = useState(false)

  useEffect(() => {
    let r1 = 0
    let r2 = 0
    let to: ReturnType<typeof setTimeout> | undefined
    let tf: ReturnType<typeof setTimeout> | undefined

    if (open) {
      setMounted(true)
      r1 = requestAnimationFrame(() => {
        r2 = requestAnimationFrame(() => setShown(true))
      })
      tf = setTimeout(() => setShown(true), 40)
    } else {
      setShown(false)
      to = setTimeout(() => setMounted(false), ms)
    }

    return () => {
      cancelAnimationFrame(r1)
      cancelAnimationFrame(r2)
      clearTimeout(to)
      clearTimeout(tf)
    }
  }, [open, ms])

  return { mounted, shown }
}
