import { useCallback, useEffect, useRef, useState } from "react"

const DISMISS_THRESHOLD = 72

export function useSwipeDismiss(
  onClose: () => void,
  enabled = true,
  resetKey?: boolean
) {
  const [dragY, setDragY] = useState(0)
  const [dragging, setDragging] = useState(false)
  const startYRef = useRef(0)
  const dragYRef = useRef(0)
  const activeRef = useRef(false)
  const scrollRef = useRef<HTMLDivElement | null>(null)

  const reset = useCallback(() => {
    dragYRef.current = 0
    setDragY(0)
    setDragging(false)
    activeRef.current = false
  }, [])

  useEffect(() => {
    if (resetKey) reset()
  }, [resetKey, reset])

  const getTransform = useCallback(
    (shown: boolean) => {
      if (dragY > 0) return `translateY(${dragY}px)`
      return shown ? "translateY(0)" : "translateY(102%)"
    },
    [dragY]
  )

  const onPointerDown = useCallback(
    (e: React.PointerEvent<HTMLElement>) => {
      if (!enabled) return
      if ((e.target as HTMLElement).closest("button")) return
      const scrollEl = scrollRef.current
      if (
        scrollEl &&
        scrollEl.scrollTop > 0 &&
        scrollEl.contains(e.target as Node)
      ) {
        return
      }
      activeRef.current = true
      startYRef.current = e.clientY
      setDragging(true)
      e.currentTarget.setPointerCapture(e.pointerId)
    },
    [enabled]
  )

  const onPointerMove = useCallback((e: React.PointerEvent<HTMLElement>) => {
    if (!activeRef.current) return
    const dy = Math.max(0, e.clientY - startYRef.current)
    dragYRef.current = dy
    setDragY(dy)
  }, [])

  const finishDrag = useCallback(
    (e: React.PointerEvent<HTMLElement>) => {
      if (!activeRef.current) return
      activeRef.current = false
      setDragging(false)
      if (e.currentTarget.hasPointerCapture(e.pointerId)) {
        e.currentTarget.releasePointerCapture(e.pointerId)
      }
      if (dragYRef.current >= DISMISS_THRESHOLD) {
        onClose()
        return
      }
      dragYRef.current = 0
      setDragY(0)
    },
    [onClose]
  )

  const pointerHandlers = {
    onPointerDown,
    onPointerMove,
    onPointerUp: finishDrag,
    onPointerCancel: finishDrag,
  }

  const panelMotionStyle = {
    transform: undefined as string | undefined,
    transition: dragging
      ? "none"
      : "transform .34s cubic-bezier(.32,.72,0,1)",
  }

  return {
    scrollRef,
    dragging,
    pointerHandlers,
    getTransform,
    panelMotionStyle,
    reset,
  }
}
